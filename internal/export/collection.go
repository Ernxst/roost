package export

import (
	"fmt"
	"io"
	"strings"

	"roost/internal/store"
)

// Collection writes one collection as a single Markdown file to share. It
// links to X and to remote images, so it reads anywhere without roost's
// attachments.
func Collection(w io.Writer, c store.Collection, bookmarks []store.Bookmark, highlights func(tweetID string) ([]store.Highlight, error)) error {
	var md strings.Builder
	md.WriteString("# " + c.Name + "\n\n")
	posts := "posts"
	if len(bookmarks) == 1 {
		posts = "post"
	}
	md.WriteString(fmt.Sprintf("%d %s from X", len(bookmarks), posts))
	if c.Query != "" {
		md.WriteString(" matching `" + c.Query + "`")
	}
	md.WriteString(", kept in roost.\n")

	for _, b := range bookmarks {
		t := b.Tweet
		md.WriteString("\n---\n\n")
		md.WriteString("**" + t.AuthorName + "** ([@" + t.AuthorHandle + "](https://x.com/" + t.AuthorHandle + ")) · " + t.CreatedAt.Format("2 Jan 2006") + "\n\n")
		md.WriteString(plainText(t) + "\n")
		for _, m := range t.Media {
			md.WriteString("\n![](" + m.Thumb + "?name=small)\n")
		}
		if q := t.Quoted; q != nil {
			md.WriteString("\n> **" + q.AuthorName + "** (@" + q.AuthorHandle + ")\n")
			for _, line := range strings.Split(plainText(*q), "\n") {
				md.WriteString("> " + line + "\n")
			}
		}
		if b.Note != "" {
			md.WriteString("\n*Note:* " + b.Note + "\n")
		}
		if hs, err := highlights(t.ID); err == nil {
			for _, h := range hs {
				md.WriteString("\n> " + strings.ReplaceAll(h.Quote, "\n", "\n> ") + "\n")
				if h.Note != "" {
					md.WriteString("\n" + h.Note + "\n")
				}
			}
		}
		md.WriteString("\n[View on X](" + t.URL + ")\n")
	}
	_, err := io.WriteString(w, md.String())
	return err
}
