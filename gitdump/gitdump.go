package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chengongpp/purge/gitdump/gin"
	"golang.org/x/net/proxy"
)

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"

type HttpClient struct {
	client *http.Client
	ua     string
}

func NewHttpClient(socksProxyAddr *string, socksUser *string, socksPass *string) (*HttpClient, error) {
	var dialer proxy.Dialer
	var err error
	// Setup SOCKS5 proxy context if provided
	// TODO proxy not completed
	if *socksProxyAddr != "" {
		dialer, err = proxy.SOCKS5("tcp", *socksProxyAddr, &proxy.Auth{User: *socksUser, Password: *socksPass}, proxy.Direct)
		if err != nil {
			log.Fatalf("Failed to create proxy due to error: %v", err)
		}
		tr := &http.Transport{
			Dial: dialer.Dial,
		}
		return &HttpClient{
			client: &http.Client{Transport: tr},
		}, nil
	} else {
		// Use default HTTP client
		dialer = proxy.Direct
		return &HttpClient{
			client: http.DefaultClient,
		}, nil
	}
}

func main() {
	savePath := flag.String("o", "", "path to save the git dump, will autometically creates a folder")
	target := flag.String("u", "", "target git disclosure URL e.g. http://example.com/.git")
	threads := flag.Int("t", 5, "downloading threads, 5 by default")
	socks := flag.String("socks", "", "SOCKS5 proxy e.g. 127.0.0.1:1145")
	socksuser := flag.String("socksuser", "", "SOCKS5 proxy username")
	sockspass := flag.String("sockspass", "", "SOCKS5 proxy password")
	if *target == "" {
		flag.PrintDefaults()
		return
	}

	path := *savePath
	if path == "" {
		// current path
		x, err := os.Executable()
		if err != nil {
			log.Fatalf("Failed to create dump folder due to error: %v", err)
		}
		path = filepath.Dir(x)
	}
	currentTime := time.Now()
	path = path + "/" + currentTime.Format("20060102_150405")
	err := os.Mkdir(path, 0755)
	if err != nil {
		log.Fatalf("Failed to create dump folder due to error: %v", err)
	}

	// Setup HTTP client
	client, err := NewHttpClient(socks, socksuser, sockspass)
	if err != nil {
		log.Fatalf("Failed to create HTTP client due to error: %v", err)
	}

	// Fetch the index file
	rsp, err := client.Download(*target + "/index")
	if err != nil {
		log.Fatalf("Failed to fetch index file due to error: %v", err)
	}
	// Parse the index file
	entries, err := ParseIndex(rsp, *target)
	if err != nil {
		log.Fatalf("Failed to parse index file due to error: %v", err)
	}
	// Fetch the objects
	pool := NewPool(client, *threads)
	pool.Download(entries)
}

func ParseIndex(indexContent []byte, baseURL string) (filenameAndUrls [][2]string, err error) {
	for r := range gin.ParseIndexContent(indexContent) {
		switch r.(type) {
		case gin.Entry:
			entry := r.(gin.Entry)
			if entry.Name == "" || entry.Sha1 == "" {
				continue
			}
			filenameAndUrls = append(filenameAndUrls, [2]string{entry.Name, baseURL + "/objects/" + entry.Sha1[:2] + "/" + entry.Sha1[2:]})
		default:
			continue
		}
	}
	return
}

func (c *HttpClient) Download(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil
	}
	// Save the index file
	rsp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

type Pool struct {
	client  *HttpClient
	workers int
}

func NewPool(client *HttpClient, workers int) *Pool {
	return &Pool{
		client:  client,
		workers: workers,
	}
}

func (p *Pool) Download(filenameAndUrls [][2]string) {
	var wg sync.WaitGroup
	jobChan := make(chan [2]string)

	for i := 0; i < p.workers; i++ {
		go p.worker(jobChan, &wg)
	}

	wg.Add(len(filenameAndUrls))
	for _, filenameAndUrl := range filenameAndUrls {
		jobChan <- filenameAndUrl
	}
	close(jobChan)
	wg.Wait()
}

func (p *Pool) worker(jobChan <-chan [2]string, wg *sync.WaitGroup) {
	defer wg.Done()
	for fileUrl := range jobChan {
		slog.Info("Downloading", "filename", fileUrl[0], "url", fileUrl[1])
		if err := p.downloadFile(fileUrl[1], fileUrl[0]); err != nil {
			slog.Error("Error downloading", "url", fileUrl[0], "error", err)
		}
	}
}

func (p *Pool) downloadFile(url, filename string) error {
	const maxRetries = 3
	var err error
	for i := 0; i < maxRetries; i++ {
		err = p.tryDownloadFile(url, filename)
		if err == nil {
			return nil
		}
		slog.Warn("Retrying download", "url", url, "attempt", i+1, "error", err)
	}
	return fmt.Errorf("failed to download %s after %d attempts: %v", url, maxRetries, err)
}

func (p *Pool) tryDownloadFile(url, filename string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", p.client.ua)
	resp, err := p.client.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s, status code: %d", url, resp.StatusCode)
	}
	// if file exists, overwrite it
	if _, err := os.Stat(filename); err == nil {
		slog.Warn("File already exists, overwriting", "filename", filename)
		if err := os.Remove(filename); err != nil {
			return err
		}
	}
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
