package download

import (
	"fmt"
	"io"
	"sync"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"github.com/lavafroth/8gooses/pkg/resource"
	"github.com/lavafroth/8gooses/pkg/constants"

	"github.com/PuerkitoBio/goquery"
)

const (
	EPISODE = iota
	ALBUM
	ARTIST
)

type Work struct {
	Destination string
	Source string
}

var work chan Work
var Tasks = new(sync.WaitGroup)

func Traverse(tags []string, destination string, entity int) error {
	if entity == EPISODE {
		nTags := len(tags)
		directory := filepath.Join(
			destination,
			filepath.Join(tags[nTags-3:]...),
		)
		episode := tags[nTags-1]
		log.Printf("Downloading episode %s to %s", episode, directory)
		os.MkdirAll(directory, 0o700)
		return linksEach(tags, ".image img", "data-src", func(i int, link string) error {
			fragments := strings.Split(strings.Trim(link, "/"), "/")
			nFragments := len(fragments) 
			if nFragments < 3 {
				return fmt.Errorf("while parsing image episode metadata: expected image location to have at least 3 fragments: found %d", nFragments)
			}
			fragments = fragments[nFragments-3:]
			fragments[1] = "fl"
			source, err := url.JoinPath(constants.Base, fragments...)
			if err != nil {
				return fmt.Errorf("while parsing episode metadata: %q", err)
			}
			Tasks.Add(1)
			work <- Work{filepath.Join(directory, fmt.Sprintf("%d%s", i, filepath.Ext(source))), source}
			return nil
		})
	}

	return linksEach(tags, "a", "href", func(i int, link string) error {
		return Traverse(resource.Tags(link), destination, entity - 1)
	})
}

func StartJobs(coroutines uint) {
	work = make(chan Work, coroutines)
	for ; coroutines > 0; coroutines-- {
		go func() {
			for w := range work {
				if err := download(w); err != nil {
					log.Printf("warning: %q", err)
				}
				Tasks.Done()
			}
		} ()
	}
}

func download(w Work) error {
	out, err := os.Create(w.Destination)
	if err != nil {
		return err
	}
	defer out.Close()

	res, err := http.Get(w.Source)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if _, err := io.Copy(out, res.Body); err != nil {
		return err
	}
	return nil
}

func linksEach(tags []string, selector string, attribute string, eachFunc func(int, string) error) error {
	var links []string

	uri, err := resource.URL(tags)
	if err != nil {
		return err
	}
	res, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s for url: %s", res.StatusCode, res.Status, uri)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	doc.
		FindMatcher(goquery.Single(".gallery")).
		Find(selector).
		Each(func(i int, s *goquery.Selection) {
			if location, ok := s.Attr(attribute); ok {
				links = append(links, location)
			}
		})
	for i, link := range(links) {
		if err := eachFunc(i, link); err != nil {
			return err
		}
	}
	return nil
}
