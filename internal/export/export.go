// Package export writes the library as a folder of Markdown notes, one per
// bookmark plus one per collection, laid out for Obsidian and readable
// anywhere.
package export

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"roost/internal/store"
	"roost/internal/xparse"
)

// Source is what the export reads; *store.Store provides it.
type Source interface {
	All() ([]store.Bookmark, error)
	Collections() ([]store.Collection, error)
	Thread(tweetID string) ([]xparse.Tweet, error)
	Highlights(tweetID string) ([]store.Highlight, error)
	Links(tweetID string) ([]store.SavedLink, error)
}

// Result says what an export wrote.
type Result struct {
	Dir   string `json:"dir"`
	Notes int    `json:"notes"`
	Media int    `json:"media"`
}

// Write exports the library into dir/roost. cached returns the path of a
// local copy of a media URL, if there is one; those files are copied next to
// the notes so images show offline. Writing again overwrites the notes
// roost wrote before and leaves any other files alone.
func Write(dir string, src Source, cached func(url string) (string, bool)) (Result, error) {
	root := filepath.Join(dir, "roost")
	res := Result{Dir: root}
	for _, d := range []string{"Bookmarks", "Collections", "attachments"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return res, err
		}
	}
	bookmarks, err := src.All()
	if err != nil {
		return res, err
	}
	cols, err := src.Collections()
	if err != nil {
		return res, err
	}
	colName := map[int64]string{}
	for _, c := range cols {
		colName[c.ID] = c.Name
	}

	names := map[string]bool{}
	noteOf := map[string]string{}
	for _, b := range bookmarks {
		name := noteName(b.Tweet)
		if names[name] {
			name += " " + b.Tweet.ID
		}
		names[name] = true
		noteOf[b.Tweet.ID] = name
	}

	members := map[string][]string{}
	for _, b := range bookmarks {
		t := b.Tweet
		var md strings.Builder
		var colNames []string
		for _, id := range b.Collections {
			if n, ok := colName[id]; ok {
				colNames = append(colNames, n)
				members[n] = append(members[n], noteOf[t.ID])
			}
		}
		md.WriteString("---\n")
		field(&md, "author", "@"+t.AuthorHandle)
		field(&md, "author_name", t.AuthorName)
		field(&md, "url", t.URL)
		field(&md, "posted", t.CreatedAt.Format("2006-01-02"))
		field(&md, "kept", b.FirstSeenAt.Format("2006-01-02"))
		if b.RemovedAt != nil {
			field(&md, "removed_on_x", b.RemovedAt.Format("2006-01-02"))
		}
		if len(colNames) > 0 {
			md.WriteString("collections:\n")
			for _, n := range colNames {
				md.WriteString("  - " + yamlString("[["+n+"]]") + "\n")
			}
		}
		md.WriteString("tags:\n  - roost\n---\n\n")

		posts := []xparse.Tweet{t}
		if thread, err := src.Thread(t.ID); err == nil && len(thread) > 1 {
			posts = thread
		}
		for i, p := range posts {
			if i > 0 {
				md.WriteString("\n---\n\n")
			}
			if len(posts) > 1 && p.ID == t.ID {
				md.WriteString("*The kept post:*\n\n")
			}
			md.WriteString(plainText(p) + "\n")
			res.Media += writeMedia(&md, root, p.Media, cached)
			if p.Quoted != nil {
				md.WriteString("\n> **" + p.Quoted.AuthorName + "** (@" + p.Quoted.AuthorHandle + ")\n")
				for _, line := range strings.Split(plainText(*p.Quoted), "\n") {
					md.WriteString("> " + line + "\n")
				}
				md.WriteString("> [View on X](" + p.Quoted.URL + ")\n")
			}
		}

		if b.Note != "" {
			md.WriteString("\n## Note\n\n" + b.Note + "\n")
		}
		if hs, err := src.Highlights(t.ID); err == nil && len(hs) > 0 {
			md.WriteString("\n## Highlights\n\n")
			for _, h := range hs {
				md.WriteString("> " + strings.ReplaceAll(h.Quote, "\n", "\n> ") + "\n")
				if h.Source != "" {
					md.WriteString("> — [" + host(h.Source) + "](" + h.Source + ")\n")
				}
				md.WriteString("\n")
			}
		}
		if ls, err := src.Links(t.ID); err == nil {
			var lines []string
			for _, l := range ls {
				if strings.Contains(l.Expanded, "x.com/") || strings.Contains(l.Expanded, "twitter.com/") {
					continue
				}
				title := l.Display
				if l.Page != nil && l.Page.Title != "" {
					title = l.Page.Title
				}
				line := "- [" + escapeLink(title) + "](" + l.Expanded + ")"
				if l.Page != nil && l.Page.Excerpt != "" {
					line += " — " + strings.ReplaceAll(l.Page.Excerpt, "\n", " ")
				}
				lines = append(lines, line)
			}
			if len(lines) > 0 {
				md.WriteString("\n## Links\n\n" + strings.Join(lines, "\n") + "\n")
			}
		}
		if err := os.WriteFile(filepath.Join(root, "Bookmarks", noteOf[t.ID]+".md"), []byte(md.String()), 0o644); err != nil {
			return res, err
		}
		res.Notes++
	}

	for _, c := range cols {
		var md strings.Builder
		md.WriteString("---\ntags:\n  - roost/collection\n")
		if c.Query != "" {
			field(&md, "search", c.Query)
		}
		md.WriteString("---\n\n")
		if c.Query != "" {
			md.WriteString("A smart collection in roost; it holds whatever matches its search.\n")
		}
		for _, n := range members[c.Name] {
			md.WriteString("- [[" + n + "]]\n")
		}
		if err := os.WriteFile(filepath.Join(root, "Collections", safeName(c.Name)+".md"), []byte(md.String()), 0o644); err != nil {
			return res, err
		}
		res.Notes++
	}
	return res, nil
}

func writeMedia(md *strings.Builder, root string, media []xparse.Media, cached func(string) (string, bool)) int {
	copied := 0
	for _, m := range media {
		url := m.Thumb + "?name=small"
		if m.Video != "" {
			if path, ok := cached(m.Video); ok {
				if name, err := copyAttachment(root, path); err == nil {
					md.WriteString("\n![[" + name + "]]\n")
					copied++
					continue
				}
			}
			url = m.Thumb
		}
		if path, ok := cached(url); ok {
			if name, err := copyAttachment(root, path); err == nil {
				md.WriteString("\n![[" + name + "]]\n")
				copied++
				continue
			}
		}
		md.WriteString("\n![](" + url + ")\n")
	}
	return copied
}

// copyAttachment copies a cached media file into the attachments folder
// unless it is already there; cache names are content-stable hashes.
func copyAttachment(root, path string) (string, error) {
	name := filepath.Base(path)
	dst := filepath.Join(root, "attachments", name)
	if _, err := os.Stat(dst); err == nil {
		return name, nil
	}
	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	return name, out.Close()
}

var (
	tcoLink = regexp.MustCompile(`https://t\.co/\w+`)
	anyLink = regexp.MustCompile(`https?://\S+`)
)

// plainText is the post's text with HTML entities decoded and t.co links
// replaced by where they lead.
func plainText(t xparse.Tweet) string {
	text := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(t.Text)
	expanded := map[string]string{}
	for _, l := range t.Links {
		expanded[l.Short] = l.Expanded
	}
	return tcoLink.ReplaceAllStringFunc(text, func(s string) string {
		if e, ok := expanded[s]; ok {
			return e
		}
		return s
	})
}

// noteName is "@handle – the first words".
func noteName(t xparse.Tweet) string {
	words := strings.Fields(anyLink.ReplaceAllString(plainText(t), ""))
	if len(words) > 8 {
		words = words[:8]
	}
	name := "@" + t.AuthorHandle
	if len(words) > 0 {
		name += " – " + strings.Join(words, " ")
	} else {
		name += " " + t.ID
	}
	return safeName(name)
}

var unsafeChars = regexp.MustCompile(`[\\/:*?"<>|#^\[\]\x00-\x1f]+`)

func safeName(s string) string {
	s = strings.TrimSpace(unsafeChars.ReplaceAllString(s, " "))
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 90 {
		s = strings.TrimSpace(string(r[:90]))
	}
	return strings.TrimRight(s, ".")
}

func field(md *strings.Builder, key, value string) {
	md.WriteString(key + ": " + yamlString(value) + "\n")
}

func yamlString(s string) string {
	return fmt.Sprintf("%q", s)
}

func escapeLink(s string) string { return strings.NewReplacer("[", "(", "]", ")").Replace(s) }

func host(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	return strings.TrimPrefix(strings.SplitN(u, "/", 2)[0], "www.")
}
