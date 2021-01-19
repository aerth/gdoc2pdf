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

var Version = "v0.0.2"

var needDocLink = "" +
	"link to google docs document is required, for example:\n" +
	"gdoc2pdf https://docs.google.com/document/d/123456...789/view"

func main() {
	log.SetFlags(0)
	var (
		proxyflag   = flag.String("proxy", "", "example: socks5://127.0.0.1:1080")
		forceFlag   = flag.Bool("f", false, "overwrite existing files")
		outputName  = flag.String("o", "", "output filename (derived from document if blank)")
		versionFlag = flag.Bool("version", false, "print version and exit")
		renameFlag  = flag.Bool("a", false, "archive mode, will save to disk without rewriting")
	)

	flag.Parse()

	if *versionFlag {
		log.Println("gdoc2pdf " + Version)
		log.Println("source: https://github.com/aerth/gdoc2pdf")
		os.Exit(0)
	}
	args := flag.Args()
	if len(args) == 0 {
		var doclink = ""
		fmt.Fprintf(os.Stderr, "Enter a google document link: ")
		fmt.Scanf("%s", &doclink)
		fmt.Fprintf(os.Stderr, "Is this accurate?            %q\nPress Y or N: ", doclink)
		yesno := ""
		fmt.Scanf("%s", &yesno)
		if strings.HasPrefix(strings.ToLower(yesno), "y") {
			args = []string{doclink}
		}
	}

	// use proxy if --proxy is set
	httpclient := tgun.Client{
		Proxy:     *proxyflag,
		UserAgent: "gdoc2pdf/" + Version,
	}
	for documentNumber, documentURL := range args {
		// fetch document ID
		u, err := url.Parse(documentURL)
		if err != nil {
			log.Fatalf("error parsing arg %d: %v", documentNumber+1, err)
		}
		paths := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(paths) != 4 {
			log.Fatalln(needDocLink)

		}
		/*	if paths[0] != "document" && paths[0] != "spreadsheets" {
				log.Println("need '/document' in URL prefix")
				log.Fatalln(needDocLink)
			}
		*/
		if paths[1] != "d" {
			log.Println("need '/document/d/' in URL prefix")
			log.Fatalln(needDocLink)
		}
		if paths[3] != "" && !(paths[3] == "edit" || paths[3] == "view" || paths[3] == "copy") {
			log.Println("need '/edit', '/view', or '/copy' in URL suffix")
			log.Fatalln(needDocLink)
		}

		// get filename if -o flag not set
		var filename string
		if *outputName != "" {
			filename = *outputName
		} else {
			// initial GET for title
			filename, err = fetchFileName(httpclient, documentURL)
			if err != nil {
				log.Fatalln("error fetching title:", err)
			}
			//		filename += ".pdf"
			log.Println("Parsed title:", filename+".pdf")
		}

		// check if filename exists
		_, err = os.Open(filename + ".pdf")
		if err == nil && !*forceFlag && !*renameFlag {
			log.Fatalln("filename already exists and -f flag not used. not overwriting. try -a to automatically save with suffix.")
		}
		if err == nil && *renameFlag {
			log.Println("found existing filename. will add suffix")
			for try := 1; try > 0; try++ {
				name := fmt.Sprintf("%s.%d.pdf", filename, try)
				fmt.Fprintf(os.Stderr, "..%d.", try)
				if _, err := os.Open(name); err == nil {
					continue
				}
				filename = name
				break
			}
			fmt.Fprintf(os.Stderr, " ")
		}

		if *outputName != "" || !strings.HasSuffix(filename, ".pdf") {
			log.Println("adding suffix pdf", filename)
			filename += ".pdf"
		}
		log.Printf("Saving PDF document: %q", filename)
		// build link
		link := fmt.Sprintf("https://docs.google.com/%s/"+
			"export?format=pdf&id=%s&includes_info_params=false", paths[0], paths[2])

		// start fetch pdf
		resp, err := httpclient.Get(link)
		if err != nil {
			log.Fatalln("fetching URL:", err)
		}
		defer resp.Body.Close()
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
			f.Close()
			log.Fatalln("error downloading pdf:", err)
		}
		f.Close()
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
