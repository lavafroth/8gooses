package download

import (
	"archive/zip"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lavafroth/8gooses/pkg/constants"
	"github.com/lavafroth/8gooses/pkg/resource"

	"github.com/PuerkitoBio/goquery"
)

const (
	EPISODE = iota
	ALBUM
	ARTIST
)

var work chan *Page
var results chan *Page
var Tasks = new(sync.WaitGroup)

type Episode struct {
	Name      string
	Directory string
	File      *os.File
	ZipWriter *zip.Writer
	Pieces    int
}

type Page struct {
	Parent *Episode
	Source string
	// This is the destination name of the file inside the zip.
	// Not to be confused with the name of the zip file itself.
	Destination string
	Body        io.ReadCloser
}

func NewPage(parent *Episode, source string, index int) *Page {
	return &Page{
		parent, source,
		fmt.Sprintf("%05d%s", index+1, filepath.Ext(source)),
		nil,
	}
}

func Traverse(tags []string, destination string, entity int) error {
	if entity == EPISODE {
		nTags := len(tags)
		name := strings.Trim(tags[nTags-1], "/")
		directory := filepath.Join(destination, tags[nTags-3], tags[nTags-2])
		filename := fmt.Sprintf("%s.cbz", name)

		os.MkdirAll(directory, 0o700)
		file, err := os.Create(filepath.Join(directory, filename))
		if err != nil {
			return fmt.Errorf("while creating %s at %d: %q", filename, directory, err)
		}

		zipWriter := zip.NewWriter(file)
		links, err := linksFor(tags, ".image img", "data-src")
		if err != nil {
			return fmt.Errorf("while parsing episode metadata: %q", err)
		}
		log.WithFields(log.Fields{
			"name":        name,
			"destination": directory,
		}).Info("Downloading episode")
		episode := Episode{name, directory, file, zipWriter, len(links)}
		for i, link := range links {
			fragments := strings.Split(strings.Trim(link, "/"), "/")
			nFragments := len(fragments)
			if nFragments < 3 {
				return fmt.Errorf("while parsing image episode metadata: expected image location to have at least 3 fragments: found %d", nFragments)
			}
			fragments = fragments[nFragments-3:]
			fragments[1] = "fl"
			source, err := url.JoinPath(constants.Base, fragments...)
			if err != nil {
				return fmt.Errorf("while joining URL base with fragments %+v: %q", fragments, err)
			}
			Tasks.Add(1)
			work <- NewPage(&episode, source, i)
		}
		return nil
	}
	entityRepr := "artist"
	if entity == ALBUM {
		entityRepr = "album"
	}

	links, err := linksFor(tags, "a", "href")
	if err != nil {
		return fmt.Errorf("while traversing links for %s: %q", entityRepr, err)
	}
	for _, link := range links {
		if err := Traverse(resource.Tags(link), destination, entity-1); err != nil {
			return fmt.Errorf("while traversing links for %s: %q", entityRepr, err)
		}
	}
	return nil
}

func WriteToCBZ() {
	for r := range results {
		if r.Body != nil {
			writer, err := r.Parent.ZipWriter.Create(r.Destination)
			if err != nil {
				log.WithFields(log.Fields{
					"artifact": r.Destination,
					"archive":  r.Parent.File.Name(),
				}).Warnf("Failed creating artifact in cbz archive: %s", err)
			}
			if _, err := io.Copy(writer, r.Body); err != nil {
				log.WithFields(log.Fields{
					"artifact": r.Destination,
					"archive":  r.Parent.File.Name(),
				}).Warnf("Failed writing to artifact in cbz archive: %s", err)
			}
			r.Body.Close()
		}
		r.Parent.Pieces--
		if r.Parent.Pieces == 0 {
			r.Parent.ZipWriter.Close()
			r.Parent.File.Close()
		}
		Tasks.Done()
	}
}

func StartJobs(coroutines uint) {
	work = make(chan *Page, coroutines)
	results = make(chan *Page)
	for ; coroutines > 0; coroutines-- {
		go func() {
			for page := range work {
				res, err := mustGet(page.Source)
				if err != nil {
					log.WithFields(log.Fields{
						"page":    page.Destination,
						"episode": page.Parent.Name,
					}).Warnf("Failed fetching page for episode: %q", err)
					results <- page
					continue
				}
				page.Body = res.Body
				results <- page
			}
		}()
	}
	go WriteToCBZ()
}

func mustGet(uri string) (*http.Response, error) {
	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s for url: %s", res.StatusCode, res.Status, uri)
	}
	return res, nil
}

func linksFor(tags []string, selector string, attribute string) ([]string, error) {
	var links []string

	uri, err := resource.URL(tags)
	if err != nil {
		return nil, err
	}

	res, err := mustGet(uri)
	if err != nil {
		return nil, err
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
