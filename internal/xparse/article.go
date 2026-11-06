package xparse

import (
	"fmt"
	"strings"
)

// Article is a long-form X article attached to a post.
type Article struct {
	// URL is the article's address as the post links to it, which is
	// where roost keeps its saved copy.
	URL     string `json:"url"`
	Title   string `json:"title"`
	Preview string `json:"preview"`
	// Text is the whole article as plain paragraphs separated by blank
	// lines. X sends it only when it shows the post on its own, so posts
	// read from the bookmarks timeline leave it empty.
	Text string `json:"-"`
}

func (t Tweet) hasArticleText() bool { return t.Article != nil && t.Article.Text != "" }

// parseArticle reads the article post r publishes into t. The link to the
// article leaves the text, as it does with attached media, and a post
// without one gains it in its links, so the saved copy of the article sits
// with the post's other linked pages.
func parseArticle(r map[string]any, t *Tweet) {
	a, _ := dig(r, "article", "article_results", "result").(map[string]any)
	id := str(a["rest_id"])
	if id == "" {
		return
	}
	t.Article = &Article{
		Title:   strings.TrimSpace(str(a["title"])),
		Preview: strings.TrimSpace(str(a["preview_text"])),
		Text:    articleText(dig(a, "content_state", "blocks")),
	}
	for _, l := range t.Links {
		if strings.HasSuffix(l.Expanded, "/i/article/"+id) {
			t.Article.URL = l.Expanded
			t.Text = strings.TrimSpace(strings.TrimSuffix(t.Text, l.Short))
			return
		}
	}
	t.Article.URL = "https://x.com/i/article/" + id
	t.Links = append(t.Links, Link{Expanded: t.Article.URL, Display: "x.com/i/article/" + id})
}

// articleText flattens the blocks of X's article editor into paragraphs.
// Images and embedded posts sit in atomic blocks with no text of their own,
// so they drop out.
func articleText(blocks any) string {
	var paras []string
	n := 0
	for _, b := range list(blocks) {
		text := strings.TrimSpace(str(dig(b, "text")))
		kind := str(dig(b, "type"))
		if kind == "ordered-list-item" {
			n++
		} else {
			n = 0
		}
		if text == "" || kind == "atomic" {
			continue
		}
		switch kind {
		case "unordered-list-item":
			text = "• " + text
		case "ordered-list-item":
			text = fmt.Sprintf("%d. %s", n, text)
		}
		paras = append(paras, text)
	}
	return strings.Join(paras, "\n\n")
}
