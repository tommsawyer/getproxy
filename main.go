package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
)

var htmlTableRowWithIPAndPort = regexp.MustCompile(`<td>(\d+\.\d+\d\.\d+\.\d+)</td><td>(\d+)`)

func main() {
	proxy, err := getProxy()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot get proxy: %v", err)
		os.Exit(1)
	}

	fmt.Println(proxy)
}

func getProxy() (*url.URL, error) {
	urls, err := parseFreeProxyList()
	if err != nil {
		return nil, err
	}

	ctx, cancelProxyChecks := context.WithCancel(context.Background())
	availableProxy := make(chan *url.URL)

	var proxyChecks sync.WaitGroup
	for _, u := range urls {
		proxyChecks.Add(1)

		go func(u *url.URL) {
			defer proxyChecks.Done()

			if isProxyAvailable(ctx, u) {
				select {
				case availableProxy <- u:
				default:
				}
			}
		}(u)
	}

	select {
	case <-allChecksFinished(&proxyChecks):
		return nil, fmt.Errorf("all proxies unavailable")
	case proxy := <-availableProxy:
		cancelProxyChecks()
		proxyChecks.Wait()
		close(availableProxy)
		return proxy, nil
	}
}

func parseFreeProxyList() ([]*url.URL, error) {
	resp, err := http.Get("https://free-proxy-list.net")
	if err != nil {
		return nil, fmt.Errorf("cannot get free proxy list: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rows := htmlTableRowWithIPAndPort.FindAllStringSubmatch(string(body), -1)
	urls := make([]*url.URL, len(rows))
	for i, row := range rows {
		ip := row[1]
		port := row[2]
		urls[i], _ = url.Parse(fmt.Sprintf("http://%s:%s", ip, port))
	}

	return urls, nil
}

func isProxyAvailable(ctx context.Context, proxy *url.URL) bool {
	withProxy := &http.Transport{Proxy: http.ProxyURL(proxy)}
	client := &http.Client{Transport: withProxy}

	req, _ := http.NewRequest("HEAD", "https://www.google.com", nil)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func allChecksFinished(checks *sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		checks.Wait()
		close(done)
	}()

	return done
}
