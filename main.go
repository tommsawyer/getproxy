package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
)

func main() {
	urls, err := parseFreeProxyList()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
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
		fmt.Fprintf(os.Stderr, "all proxies unavailable")
		os.Exit(1)
	case proxy := <-availableProxy:
		cancelProxyChecks()
		proxyChecks.Wait()
		fmt.Println(proxy)
	}
}

func isProxyAvailable(ctx context.Context, proxy *url.URL) bool {
	transport := &http.Transport{Proxy: http.ProxyURL(proxy)}
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("GET", "https://google.com", nil)

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return false
	}
	resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func exitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func allChecksFinished(wg sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}
