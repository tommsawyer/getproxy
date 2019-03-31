package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var rowWithIpAndPort = regexp.MustCompile(`<td>\d+\.\d+\d\.\d+\.\d+</td><td>\d+`)

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

	rows := rowWithIpAndPort.FindAllString(string(bs), -1)
	urls := make([]*url.URL, len(rows))
	for i, row := range rows {
		rawURL := "http://" + strings.Replace(strings.Replace(row, "</td><td>", ":", -1), "<td>", "", -1)

		u, _ := url.Parse(rawURL)
		urls[i] = u
	}

	return urls, nil
}
