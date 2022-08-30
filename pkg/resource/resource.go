package resource

import (
	"net/url"
	"regexp"
	"github.com/lavafroth/8gooses/pkg/constants"
)

var re *regexp.Regexp

func init() {
	re = regexp.MustCompile("(https://)?(comics\\.)?(8muses\\.)?(com/)?(comics/)?((picture|album)/)?(?P<artist>[A-Za-z0-9\\-]+/?)(?P<album>[A-Za-z0-9\\-]+/?)?(?P<episode>[A-Za-z0-9\\-]+/?)?")
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

// URL uses the tags for a resource and reconstructs the full URL
func URL(tags []string) (string, error) {
	return url.JoinPath(constants.Base, append([]string{"comics", "album"}, tags...)...)
}
