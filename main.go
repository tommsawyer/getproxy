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

var rowWithIpAndPort = regexp.MustCompile(`<td>(\d+\.\d+\d\.\d+\.\d+)</td><td>(\d+)`)

func main() {
	urls, err := parseFreeProxyList()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	ctx, cancelProxyChecks := context.WithCancel(context.Background())
	availableProxy := make(chan *url.URL)

	var proxyChecks sync.WaitGroup
	for _, u := range urls {
		proxyChecks.Add(1)

		go func(u *url.URL) {
			defer proxyChecks.Done()

			if isProxyAvailable(ctx, u) {
				availableProxy <- u
			}
		}(u)
	}

	select {
	case <-allChecksFinished(proxyChecks):
		fmt.Fprintln(os.Stderr, "all proxies unavailable")
		os.Exit(1)
	case proxy := <-availableProxy:
		cancelProxyChecks()
		proxyChecks.Wait()
		fmt.Println(proxy)
	}
}

func parseFreeProxyList() ([]*url.URL, error) {
	resp, err := http.Get("https://free-proxy-list.net")
	if err != nil {
		return nil, fmt.Errorf("cannot get free proxy list: %v", err)
	}
	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rows := rowWithIpAndPort.FindAllStringSubmatch(string(bs), -1)
	urls := make([]*url.URL, len(rows))
	for i, row := range rows {
		u, _ := url.Parse(fmt.Sprintf("http://%s:%s", row[1], row[2]))
		urls[i] = u
	}

	return urls, nil
}

func isProxyAvailable(ctx context.Context, proxy *url.URL) bool {
	transport := &http.Transport{Proxy: http.ProxyURL(proxy)}
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("HEAD", "https://www.google.com", nil)

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func allChecksFinished(wg sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}
