package store

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Duplicates is a set of bookmarks that point at the same thing.
type Duplicates struct {
	// Reason is "post" for bookmarks of or quoting the same post, "link"
	// for the same page and "image" for the same picture.
	Reason string `json:"reason"`
	// Key is the post ID, normalised URL or image URL they share.
	Key       string     `json:"key"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

// Wall returns every bookmark with photos, videos or GIFs, newest first.
func (s *Store) Wall() ([]Bookmark, error) {
	return s.selectBookmarks("t.has_media = 1", nil, "ORDER BY b.sort_index DESC")
}

// FindDuplicates groups bookmarks that share a post, a link or an image.
// A bookmark joins at most one group per reason, and groups that hold the
// same bookmarks as an earlier group are left out.
func (s *Store) FindDuplicates() ([]Duplicates, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	type group struct {
		reason, key string
		members     []int
	}
	var groups []*group
	byKey := map[string]*group{}
	add := func(reason, key string, i int) {
		if key == "" {
			return
		}
		g := byKey[reason+" "+key]
		if g == nil {
			g = &group{reason: reason, key: key}
			byKey[reason+" "+key] = g
			groups = append(groups, g)
		}
		if n := len(g.members); n == 0 || g.members[n-1] != i {
			g.members = append(g.members, i)
		}
	}
	for i, b := range all {
		t := b.Tweet
		add("post", t.ID, i)
		if t.Quoted != nil {
			add("post", t.Quoted.ID, i)
		}
		for _, l := range t.Links {
			if id := statusID(l.Expanded); id != "" {
				add("post", id, i)
			} else {
				add("link", NormaliseURL(l.Expanded), i)
			}
		}
		for _, m := range t.Media {
			add("image", m.Thumb, i)
		}
	}

	seen := map[string]bool{}
	var out []Duplicates
	for _, g := range groups {
		if len(g.members) < 2 {
			continue
		}
		sig := make([]string, len(g.members))
		for j, i := range g.members {
			sig[j] = all[i].Tweet.ID
		}
		sort.Strings(sig)
		if k := strings.Join(sig, ","); seen[k] {
			continue
		} else {
			seen[k] = true
		}
		d := Duplicates{Reason: g.reason, Key: g.key}
		for _, i := range g.members {
			d.Bookmarks = append(d.Bookmarks, all[i])
		}
		out = append(out, d)
	}
	return out, nil
}

var statusPath = regexp.MustCompile(`^/(?:[^/]+|i(?:/web)?)/status(?:es)?/(\d+)`)

// statusID returns the ID of the post a link to X points at.
func statusID(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") {
	case "x.com", "twitter.com", "mobile.twitter.com", "mobile.x.com":
	default:
		return ""
	}
	if m := statusPath.FindStringSubmatch(u.Path); m != nil {
		return m[1]
	}
	return ""
}

// trackingParams are query parameters that say where a click came from
// rather than what it points at.
var trackingParams = map[string]bool{
	"fbclid": true, "gclid": true, "mc_cid": true, "mc_eid": true, "ref": true, "ref_src": true,
	"ref_url": true, "s": true, "si": true, "igshid": true, "source": true, "smid": true,
}

// NormaliseURL reduces a URL to the page it names: no scheme, no www, no
// fragment, no tracking parameters and no trailing slash.
func NormaliseURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	q := u.Query()
	for k := range q {
		if trackingParams[strings.ToLower(k)] || strings.HasPrefix(strings.ToLower(k), "utm_") {
			q.Del(k)
		}
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if host == "youtu.be" {
		host, q = "youtube.com", url.Values{"v": {strings.TrimPrefix(u.Path, "/")}}
		u.Path = "/watch"
	}
	out := host + strings.TrimRight(u.EscapedPath(), "/")
	if enc := q.Encode(); enc != "" {
		out += "?" + enc
	}
	return out
}
