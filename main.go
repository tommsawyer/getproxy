package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	resp, err := http.Get("https://free-proxy-list.net")
	if err != nil {
		exitWithError("cannot get free proxy list: %v", err)
	}
	defer resp.Body.Close()

	freeProxyList, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		exitWithError("cannot read document body: %v", err)
	}

	urls := parseFreeProxyList(freeProxyList)

	ctx, cancelAllProxyChecks := context.WithCancel(context.Background())

	availableProxy := make(chan *url.URL)
	allProxiesUnavailable := make(chan struct{})

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

	go func() {
		proxyChecks.Wait()
		allProxiesUnavailable <- struct{}{}
	}()

	select {
	case <-allProxiesUnavailable:
		exitWithError("all proxies unavailable")
	case proxy := <-availableProxy:
		cancelAllProxyChecks()
		proxyChecks.Wait()
		fmt.Println(proxy)
	}
}

func parseFreeProxyList(doc *goquery.Document) []*url.URL {
	rows := doc.Find(".table-responsive tbody > tr")
	urls := make([]*url.URL, rows.Length())

	rows.Each(func(i int, row *goquery.Selection) {
		ip := getCellText(row, 0)
		port := getCellText(row, 1)

		u, err := url.Parse("http://" + ip + ":" + port)
		if err != nil {
			return
		}

		urls[i] = u
	})

	return urls
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

func getCellText(row *goquery.Selection, idx int) string {
	return row.Find("td").Eq(idx).Text()
}

func exitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
