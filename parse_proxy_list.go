package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

const freeProxyListSite = "https://free-proxy-list.net"

func parseFreeProxyList() ([]*url.URL, error) {
	resp, err := http.Get(freeProxyListSite)
	if err != nil {
		return nil, fmt.Errorf("cannot get free proxy list: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read document body: %v", err)
	}

	rows := doc.Find(".table-responsive tbody > tr")
	urls := make([]*url.URL, 0, rows.Length())

	rows.Each(func(i int, row *goquery.Selection) {
		u, err := parseURL(row)
		if err != nil {
			return
		}

		urls = append(urls, u)
	})

	return urls, nil
}

func parseURL(row *goquery.Selection) (*url.URL, error) {
	ip := getCellText(row, 0)
	port := getCellText(row, 1)
	rawURL := fmt.Sprintf("http://%s:%s", ip, port)

	return url.Parse(rawURL)
}

func getCellText(row *goquery.Selection, idx int) string {
	return row.Find("td").Eq(idx).Text()
}
