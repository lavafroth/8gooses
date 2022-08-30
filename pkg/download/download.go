package download

import (
	"fmt"
	"io"
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

func Artist(tags []string, destination string) error {
	links, err := linksFor(tags, "a", "href")
	if err != nil {
		return err
	}
	for _, link := range links {
		if err := Album(resource.Tags(link), destination); err != nil {
			return err
		}
	}
	return nil
}

func Album(tags []string, destination string) error {
	links, err := linksFor(tags, "a", "href")
	if err != nil {
		return err
	}
	for _, link := range links {
		if err := Episode(resource.Tags(link), destination); err != nil {
			return err
		}
	}
	return nil
}

func Episode(tags []string, destination string) error {
	directory := filepath.Join(
		destination,
		filepath.Join(tags[len(tags)-3:]...),
	)
	episode := tags[len(tags)-1]
	log.Printf("Downloading episode %s to %s", episode, directory)
	os.MkdirAll(directory, 0o700)
	links, err := linksFor(tags, ".image img", "data-src")
	if err != nil {
		return fmt.Errorf("while parsing episode metadata: %q", err)
	}
	for i, link := range links {
		fragments := strings.Split(strings.Trim(link, "/"), "/")
		if len(fragments) < 3 {
			return fmt.Errorf("while parsing image episode metadata: expected image location to have at least 3 fragments: found %d", len(fragments))
		}
		fragments = fragments[len(fragments)-3:]
		fragments[1] = "fl"
		source, err := url.JoinPath(constants.Base, fragments...)
		if err != nil {
			return fmt.Errorf("while parsing episode metadata: %q", err)
		}
		if err := download(filepath.Join(
			directory,
			fmt.Sprintf("%d%s", i, filepath.Ext(source)),
		), source); err != nil {
			return fmt.Errorf("while parsing episode metadata: %q", err)
		}

	}
	return nil
}

func download(destination string, source string) error {
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	res, err := http.Get(source)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if _, err := io.Copy(out, res.Body); err != nil {
		return err
	}
	return nil
}

func linksFor(tags []string, selector string, attribute string) ([]string, error) {
	var links []string

	uri, err := resource.URL(tags)
	if err != nil {
		return nil, err
	}
	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s for url: %s", res.StatusCode, res.Status, uri)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	doc.
		FindMatcher(goquery.Single(".gallery")).
		Find(selector).
		Each(func(i int, s *goquery.Selection) {
			if location, ok := s.Attr(attribute); ok {
				links = append(links, location)
			}
		})
	return links, nil
}
