// Copyright (c) 2020 aerth <aerth@riseup.net>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// gdoc2pdf command downloads a PDF file from a Google Docs URL
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/aerth/tgun"
	"golang.org/x/net/html"
)

var needDocLink = "" +
	"link to google docs document is required, for example:\n" +
	"gdoc2pdf https://docs.google.com/document/d/123456...789/view"

func main() {
	log.SetFlags(0)
	var (
		proxyflag  = flag.String("proxy", "", "example: socks5://127.0.0.1:1080")
		forceFlag  = flag.Bool("f", false, "overwrite existing files")
		outputName = flag.String("o", "", "output filename (derived from document if blank)")
	)

	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatalln(needDocLink)
	}

	// use proxy if --proxy is set
	httpclient := tgun.Client{
		Proxy:     *proxyflag,
		UserAgent: "gdocs2pdf/1.0",
	}
	args := flag.Args()
	for i, v := range args {
		// fetch document ID
		u, err := url.Parse(v)
		if err != nil {
			log.Fatalf("error parsing arg %d: %v", i+1, err)
		}
		paths := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(paths) != 4 {
			log.Fatalln(needDocLink)

		}
		if paths[0] != "document" {
			log.Println("need '/document' in URL prefix")
			log.Fatalln(needDocLink)
		}
		if paths[1] != "d" {
			log.Println("need '/document/d/' in URL prefix")
			log.Fatalln(needDocLink)
		}
		if !(paths[3] == "edit" || paths[3] == "view" || paths[3] == "copy") {
			log.Println("need '/edit', '/view', or '/copy' in URL suffix")
			log.Fatalln(needDocLink)
		}

		// get filename if -o flag not set
		var filename string
		if *outputName != "" {
			filename = *outputName
		} else {
			filename, err = fetchFileName(httpclient, v)
			if err != nil {
				log.Fatalln("error fetching title:", err)
			}
			filename += ".pdf"
			log.Println("Parsed title:", filename)
		}

		// check if filename exists
		if _, err := os.Open(filename); err == nil && !*forceFlag {
			log.Fatalln("filename already exists and -f flag not used. not overwriting.")
		}

		// build link
		link := fmt.Sprintf("https://docs.google.com/document/"+
			"export?format=pdf&id=%s&includes_info_params=false", paths[2])

		// start fetch pdf
		log.Printf("Downloading PDF document: %q (%q)", filename, flag.Args())
		resp, err := httpclient.Get(link)
		defer resp.Body.Close()

		if err != nil {
			log.Fatalln("fetching URL:", err)
		}
		if resp.StatusCode != 200 {
			log.Fatalf("fetching URL: %q (%d)", resp.Status, resp.StatusCode)
		}

		// save pdf
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Fatalln("couldn't create pdf file locally:", err)
		}

		n, err := io.Copy(f, resp.Body)
		if err != nil {
			log.Fatalln("error downloading pdf:", err)
		}
		log.Printf("saved PDF %q (%d bytes)", filename, n)
	}
}

// parse filename from google docs page title
// (requires a second HTTP Request)
func fetchFileName(h tgun.Client, link string) (string, error) {
	var title string
	resp, err := h.Get(link)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return "", fmt.Errorf("parsing title returned error: %v", tt.String())
		}
		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "title" {
				z.Next()
				title = z.Token().Data
				break
			}
		}
	}
	// trim suffix
	return strings.TrimSuffix(title, " - Google Docs"), nil
}
