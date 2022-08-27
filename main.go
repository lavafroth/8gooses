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
	"strings"
)

const base string = "https://comics.8muses.com"

func DownloadAlbum(uri string) error {
	links, err := Links(uri)
	if err != nil {
		return err
	}
	for _, link := range links {
		if err := DownloadEpisode(link); err != nil {
			return err
		}
	}
	return nil
}

func DownloadEpisode(uri string) error {
	pages, directory, err := Enumerate(uri)
	if err != nil {
		return err
	}
	images, err := Images(pages)
	if err != nil {
		return err
	}
    fragments := Fragments(uri)
    episode := fragments[len(fragments)-1]
	log.Printf("Downloading episode %s to %s", episode, directory)
	DownloadAll(images, directory)
	return nil
}

func DocumentAt(uri string) (*goquery.Document, error) {
	uri, err := AbsoluteURL(uri)
	if err != nil {
		return nil, err
	}
	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
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

func Fragments(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

func Images(links []string) ([]string, error) {
	var images []string
	for _, link := range links {
		fragments := Fragments(link)
		if len(fragments) < 3 {
			return images, fmt.Errorf("while parsing image episode metadata: expected image location to have at least 3 fragments: found %d", len(fragments))
		}
		fragments = fragments[len(fragments)-3:]
		fragments[1] = "fl"
		source, err := url.JoinPath(base, fragments...)
		if err != nil {
			return images, fmt.Errorf("while parsing episode metadata: %q", err)
		}
		images = append(images, source)
	}
	return images, nil
}

func DownloadAll(images []string, directory string) error {
	for i, source := range images {
		if err := Download(filepath.Join(
			directory,
			fmt.Sprintf("%d%s", i, filepath.Ext(source)),
		), source); err != nil {
			return fmt.Errorf("while parsing episode metadata: %q", err)
		}
	}
	return nil
}

func Enumerate(uri string) ([]string, string, error) {
	var pages []string
	var directory string
	doc, err := DocumentAt(uri)
	if err != nil {
		return pages, directory, fmt.Errorf("while parsing episode metadata: %q", err)
	}
	fragments := Fragments(uri)
	fragments = fragments[len(fragments)-3:]
	directory = filepath.Join(fragments...)
	os.MkdirAll(directory, 0o700)
	doc.
		FindMatcher(goquery.Single(".gallery")).
		Find(".image img").
		Each(func(i int, s *goquery.Selection) {
			if location, ok := s.Attr("data-src"); ok {
				pages = append(pages, location)
			}
		})
	return pages, directory, nil
}

func Links(uri string) ([]string, error) {
	var links []string
	doc, err := DocumentAt(uri)
	if err != nil {
		return links, err
	}
	doc.
		FindMatcher(goquery.Single(".gallery")).
		Find("a").
		Each(func(i int, s *goquery.Selection) {
			if location, ok := s.Attr("href"); ok {
				links = append(links, location)
			}
		})
	return links, nil
}

// Tags strips a URL or partial URL to the artist, album and optionally an episode
func Tags(path string) ([]string, error) {
	// If we are given a complete URL,
	// we can trim off the base URL.
	if strings.HasPrefix(path, base) {
		path = strings.TrimPrefix(path, base)
	}
	// For the directories forming the path,
	fragments := Fragments(path)
	// if the path begins with "comics", we can trim that.
	if len(fragments) > 3 && fragments[0] == "comics" {
		fragments = fragments[2:]
	}
	// If the path begins with "album" or "picture", we can trim that too.
	if len(fragments) > 2 && (fragments[0] == "album" || fragments[0] == "picture") {
		fragments = fragments[1:]
	}
	// If after trimming, we are left with
	// less than 2 directories, something
	// has gone terribly wrong. Bail.
	if len(fragments) < 2 {
		return fragments, fmt.Errorf("failed to parse uri or path: %s", path)
	}
	// If we are left with more than 2 directories,
	// the user would have probably included the
	// page number in the path as well.
	// We can safely ignore that and download the
	// specified episode.
	if len(fragments) > 2 {
		return fragments[:3], nil
	}
	return fragments, nil
}

// AbsoluteURL finds the tags in a URI and reconstructs the full URL
func AbsoluteURL(uri string) (string, error) {
	tags, err := Tags(uri)
	if err != nil {
		return "", err
	}
	return url.JoinPath("https://comics.8muses.com/comics/album", tags...)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <URL / partial URL>\n", os.Args[0])
		return
	}
	tags, err := Tags(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	resource, err := AbsoluteURL(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	if len(tags) == 2 {
		// Download all episodes in the album
		if err := DownloadAlbum(resource); err != nil {
			log.Fatalln(err)
		}
		return
	}
	// Download one episode
	if err := DownloadEpisode(resource); err != nil {
		log.Fatalln(err)
	}
}
