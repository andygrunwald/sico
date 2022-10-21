package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

// Sitemap represents a normal sitemap xml structure
type Sitemap struct {
	URLs []URLEntry `xml:"url"`
}

// SitemapIndex represents a large or split up sitemap xml structure.
//
// See https://developers.google.com/search/docs/crawling-indexing/sitemaps/large-sitemaps
type SitemapIndex struct {
	Sitemaps []URLEntry `xml:"sitemap"`
}

// URLEntry represents a single URL entry in a sitemap
type URLEntry struct {
	Loc string `xml:"loc"`
}

type URLMap map[string]struct{}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	//originalURL := "https://andygrunwald.com/sitemap.xml"
	//newUrl := "https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/sitemap-index.xml"
	//newBaseURL := "https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/"

	originalURL := flag.String("source", "https://example.com/sitemap.xml", "Source Sitemap URL - Sitemap you want to check against")
	newUrl := flag.String("new", "https://example-new.com/sitemap.xml", "New Sitemap URL - Sitemap entries you want to check for presence")
	newBaseURL := flag.String("newBaseURL", "", "Base URL that will be used if `new` contains a SitemapIndex to replace the SitemapIndex entries")

	var excludeFlag arrayFlags
	flag.Var(&excludeFlag, "exclude", "Regex to match against URLs in {source} sitemap that don't need to be in {new} sitemap. It can be defined multiple times.")
	flag.Parse()

	excludesF := []string{
		"andygrunwald\\.com/tags/",
	}

	excludes := []*regexp.Regexp{}
	for _, s := range excludeFlag {
		excludes = append(excludes, regexp.MustCompile(s))
	}

	originalSitemapContent, err := readRemoteFile(*originalURL)
	if err != nil {
		log.Fatal(err)
	}

	originalSitemap, err := readSitemap(originalSitemapContent, "")
	if err != nil {
		log.Fatal(err)
	}

	newSitemapContent, err := readRemoteFile(*newUrl)
	if err != nil {
		log.Fatal(err)
	}

	newSitemap, err := readSitemap(newSitemapContent, *newBaseURL)
	if err != nil {
		log.Fatal(err)
	}

	newURLMap := sitemapToURLMap(*newSitemap)
	missingURLsMap := URLMap{}
	excluded := 0
	missing := 0
	urlsChecked := 0
	for _, entry := range originalSitemap.URLs {
		urlsChecked++

		skip := false
		for _, exclude := range excludes {
			skip = exclude.MatchString(entry.Loc)
			if skip {
				log.Printf("%s skipped, because of exclude '%s'", entry.Loc, exclude.String())
				excluded++
				break
			}
		}

		if skip {
			continue
		}

		if _, ok := newURLMap[entry.Loc]; !ok {
			missingURLsMap[entry.Loc] = struct{}{}
			missing++
		}
	}

	fmt.Println()
	fmt.Println("Result")
	fmt.Println("=============")
	fmt.Printf("Source Sitemap: %s\n", *originalURL)
	fmt.Printf("URLs checked (from source sitemap): %d\n", urlsChecked)
	fmt.Printf("New Sitemap: %s\n", *newUrl)
	fmt.Printf("Excludes configured: %d\n", len(excludesF))
	fmt.Println()
	fmt.Printf("URLs skipped because they matched an exclude: %d\n", excluded)
	fmt.Printf("URLs missing from source sitemap in new sitemap: %d\n", missing)
	fmt.Println()
	if missing > 0 {
		fmt.Println("Missing URLs in the new sitemap:")
		for k := range missingURLsMap {
			fmt.Println(k)
		}
	}
}

// sitemapToURLMap transform a Sitemap
// into a URLMap where the URL of a
// sitemap entry is the key.
func sitemapToURLMap(s Sitemap) URLMap {
	m := URLMap{}

	for _, entry := range s.URLs {
		m[entry.Loc] = struct{}{}
	}

	return m
}

// readSitemap parses a XML sitemap out of data.
// If data contains a sitemapindex, rather than a normal sitemap,
// this sitemapindex is parsed and retrieved and the splitted sitemaps
// merged into one big sitemap.
//
// If newBaseURL is given, sitemap urls inside a sitemap index will be
// replaced (host only).
func readSitemap(data []byte, newBaseURL string) (*Sitemap, error) {
	// If it is a large sitemap splitted into multiple sitemaps, ...
	// 	<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	// 		<sitemap>
	// 			<loc>https://andygrunwald.com/sitemap-0.xml</loc>
	// 		</sitemap>
	//		[...]
	// ...we need to read each small sitemap and merge them together
	if bytes.Contains(data, []byte("<sitemapindex")) {
		sitemapIndex := new(SitemapIndex)
		err := xml.Unmarshal(data, sitemapIndex)
		if err != nil {
			return nil, err
		}

		mergedSitemap := new(Sitemap)
		for _, sitemapURLs := range sitemapIndex.Sitemaps {
			// If the sub-sitemap is already part of the new website
			// AND the new website has the final base url already set,
			// but is part of the staging system, it might be that the
			// base URL needs to be replaced.
			// E.g.
			//	- The original website has one sitemap
			//	- The new system has a split sitemap
			newUrl, err := replaceURL(sitemapURLs.Loc, newBaseURL)
			if err != nil {
				return nil, err
			}

			sitemapContent, err := readRemoteFile(newUrl)
			if err != nil {
				return nil, err
			}

			sitemap, err := readSitemap(sitemapContent, newBaseURL)
			if err != nil {
				return nil, err
			}

			mergedSitemap.URLs = append(mergedSitemap.URLs, sitemap.URLs...)
		}

		return mergedSitemap, nil
	}

	// If it is a normal sitemap like ...
	//	<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
	//		<url>
	//			<loc>https://andygrunwald.com/</loc>
	//			<lastmod>2021-09-21T20:30:00+02:00</lastmod>
	//		</url>
	//		<url>
	//			[...]
	//		</url>
	//		[...]
	// ... we simply read it.
	sitemap := new(Sitemap)
	err := xml.Unmarshal(data, sitemap)

	return sitemap, err
}

// readRemoteFile reads u via an HTTP GET request.
func readRemoteFile(u string) ([]byte, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("get error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return data, nil
}

// replaceURL replaces URL Scheme and URL Host of u
// with URL Scheme and URL Host of new.
// All other URL elements of u stay the same (like Path, Fragment, ...)
//
// In case of an error, original u will be returned.
func replaceURL(u string, new string) (string, error) {
	if len(new) == 0 {
		return u, nil
	}

	uParsed, err := url.Parse(u)
	if err != nil {
		return u, err
	}

	newParsed, err := url.Parse(new)
	if err != nil {
		return u, err
	}

	uParsed.Scheme = newParsed.Scheme
	uParsed.Host = newParsed.Host

	return uParsed.String(), nil
}
