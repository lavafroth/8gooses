package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const base string = "https://comics.8muses.com"

var re *regexp.Regexp

func DownloadArtist(tags []string) error {
	links, err := Links(tags, "a", "href")
	if err != nil {
		return err
	}
	for _, link := range links {
		if err := DownloadAlbum(Tags(link)); err != nil {
			return err
		}
	}
	return nil
}

func DownloadAlbum(tags []string) error {
	links, err := Links(tags, "a", "href")
	if err != nil {
		return err
	}
	for _, link := range links {
		if err := DownloadEpisode(Tags(link)); err != nil {
			return err
		}
	}
	return nil
}

func DownloadEpisode(tags []string) error {
	var directory string
	directory = filepath.Join(tags[len(tags)-3:]...)
	episode := tags[len(tags)-1]
	log.Printf("Downloading episode %s to %s", episode, directory)
	os.MkdirAll(directory, 0o700)
	links, err := Links(tags, ".image img", "data-src")
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
		source, err := url.JoinPath(base, fragments...)
		if err != nil {
			return fmt.Errorf("while parsing episode metadata: %q", err)
		}
		if err := Download(filepath.Join(
			directory,
			fmt.Sprintf("%d%s", i, filepath.Ext(source)),
		), source); err != nil {
			return fmt.Errorf("while parsing episode metadata: %q", err)
		}

	}
	return nil
}

func Download(destination string, source string) error {
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

func Links(tags []string, selector string, attribute string) ([]string, error) {
	var links []string

	uri, err := AbsoluteURL(tags)
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

// Tags filters the artist, album and optionally the episode from a complete or partial URL
func Tags(path string) (tags []string) {
	groups := re.SubexpNames()
	for id, value := range re.FindAllStringSubmatch(path, -1)[0] {
		switch groups[id] {
		case "artist", "album", "episode":
			if value != "" {
				tags = append(tags, value)
			}
		}
	}
	return
}

// AbsoluteURL uses the tags for a resource and reconstructs the full URL
func AbsoluteURL(tags []string) (string, error) {
	return url.JoinPath(base, append([]string{"comics", "album"}, tags...)...)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <URL / partial URL>\n", os.Args[0])
		return
	}

	re = regexp.MustCompile("(https://)?(comics\\.)?(8muses\\.)?(com/)?(comics/)?((picture|album)/)?(?P<artist>[A-Za-z0-9\\-]+/?)(?P<album>[A-Za-z0-9\\-]+/?)?(?P<episode>[A-Za-z0-9\\-]+/?)?")

	tags := Tags(os.Args[1])

	// Default to downloading a single episode
	action := DownloadEpisode
	switch len(tags) {
	case 1:
		// Download all episodes by an artist
		action = DownloadArtist
	case 2:
		// Download all episodes in the album
		action = DownloadAlbum
	}
	if err := action(tags); err != nil {
		log.Fatalln(err)
	}
}
