package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"golang.org/x/exp/mmap"
	"golang.org/x/net/proxy"
)

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"

type HttpClient struct {
	client *http.Client
}

func NewHttpClient(socksProxyAddr *string, socksUser *string, socksPass *string) (*HttpClient, error) {
	var dialer proxy.Dialer
	var err error
	// Setup SOCKS5 proxy context if provided
	if *socksProxyAddr != "" {
		dialer, err = proxy.SOCKS5("tcp", *socksProxyAddr, &proxy.Auth{User: *socksUser, Password: *socksPass}, proxy.Direct)
		if err != nil {
			log.Fatalf("Failed to create proxy due to error: %v", err)
		}
	} else {
		// Use default HTTP client
		dialer = proxy.Direct
	}
	tr := &http.Transport{
		Dial: dialer.Dial,
	}
	return &HttpClient{
		client: &http.Client{Transport: tr},
	}, nil
}

func (c *HttpClient) FetchAsFile(url string, filename string) (bool, error) {
	fp, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file %s due to error: %v", filename, err)
		return false, err
	}
	defer fp.Close()
	retry := 0
	for retry < 3 {
		rsp, err := c.client.Get(url)
		if err != nil {
			log.Printf("Trying %d of 3, failed to fetch file due to error: %v", retry+1, err)
			retry++
			continue
		}
		defer rsp.Body.Close()
		if rsp.StatusCode != http.StatusOK {
			log.Printf("Failed to fetch %s due to status: [%d]", url, rsp.StatusCode)
			return false, nil
		}
		_, err = io.Copy(fp, rsp.Body)
		if err != nil {
			log.Printf("Failed to write file %s due to error: %v", filename, err)
			return false, err
		}
		log.Printf("[OK] Fetched: %s from %s", filename, url)
		return true, nil
	}
	return false, nil
}

func main() {
	savePath := flag.String("o", "", "path to save the git dump, will autometically creates a folder")
	target := flag.String("u", "", "target git disclosure URL e.g. http://example.com/.git")
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
	ok, err := client.FetchAsFile(*target+"/index", path+"/index")
	if err != nil || !ok {
		log.Fatalf("Failed to fetch index file due to error: %v", err)
	}
	// Parse the index file
	entries, err := ParseIndex(path + "/index")
	if err != nil {
		log.Fatalf("Failed to parse index file due to error: %v", err)
	}
	// Fetch the objects
	for _, entry := range entries {

	}
}

func ParseIndex(filename string) (*[]orderedmap.OrderedMap[string, any], error) {
	reader, err := mmap.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file due to error: %v", err)
	}
	defer reader.Close()
	log.Println("Parsing index file")
	entries := &[]orderedmap.OrderedMap[string, any]{}
	// Parse the index file
	//
	return entries, nil
}
