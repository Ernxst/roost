// Package links saves a readable copy of the pages bookmarks link to, so a
// link still reads after the page changes or disappears.
package links

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"codeberg.org/readeck/go-readability/v2"
)

// Page is what roost keeps of a linked page. Only text is kept: the page's
// own markup never reaches the app.
type Page struct {
	URL      string `json:"url"`
	FinalURL string `json:"finalUrl"`
	Title    string `json:"title"`
	Site     string `json:"site"`
	Byline   string `json:"byline"`
	Excerpt  string `json:"excerpt"`
	Text     string `json:"text,omitempty"`
	// Error says why the page could not be saved; the link is not retried.
	Error string `json:"error,omitempty"`
}

// Skip reports whether u is not worth saving: posts on X, which roost
// handles itself, and anything that is not a web page.
func Skip(u string) bool {
	p, err := url.Parse(u)
	if err != nil || (p.Scheme != "http" && p.Scheme != "https") {
		return true
	}
	switch strings.TrimPrefix(p.Hostname(), "www.") {
	case "x.com", "twitter.com", "t.co", "mobile.twitter.com":
		return true
	}
	return false
}

const maxPage = 8 << 20

var client = &http.Client{Timeout: 30 * time.Second}

// Fetch downloads u and extracts its readable text. A page that loads but
// has no readable text comes back with its title and an Error.
func Fetch(ctx context.Context, u string) Page {
	page := Page{URL: u, FinalURL: u}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		page.Error = err.Error()
		return page
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := client.Do(req)
	if err != nil {
		page.Error = err.Error()
		return page
	}
	defer resp.Body.Close()
	page.FinalURL = resp.Request.URL.String()
	page.Site = strings.TrimPrefix(resp.Request.URL.Hostname(), "www.")
	if resp.StatusCode != http.StatusOK {
		page.Error = resp.Status
		return page
	}
	if ct, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); ct != "text/html" && ct != "application/xhtml+xml" {
		page.Error = fmt.Sprintf("not a web page (%s)", ct)
		return page
	}
	article, err := readability.FromReader(io.LimitReader(resp.Body, maxPage), resp.Request.URL)
	if err != nil {
		page.Error = err.Error()
		return page
	}
	page.Title = strings.TrimSpace(article.Title())
	page.Byline = strings.TrimSpace(article.Byline())
	page.Excerpt = strings.TrimSpace(article.Excerpt())
	if site := strings.TrimSpace(article.SiteName()); site != "" {
		page.Site = site
	}
	var text strings.Builder
	if article.Node != nil && article.RenderText(&text) == nil {
		page.Text = tidy(text.String())
	}
	if len(page.Text) < 200 {
		page.Text = ""
		page.Error = "no readable text"
	}
	return page
}

var blankLines = regexp.MustCompile(`\n[ \t]*(\n[ \t]*)+`)

// tidy trims each line and collapses runs of blank lines into one.
func tidy(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.TrimSpace(blankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}
