package store

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// A search is plain words, matched as prefixes against the text, author and
// quoted post, mixed with operators:
//
//	from:handle            posts by that author
//	has:media|photo|video|gif|link|article|quote|note|thread|highlight
//	in:name, -in:name      in, or not in, a collection; in:none for none
//	is:gone|opened|unopened|aside   removed on X, opened in roost or not,
//	                       set aside while sorting
//	before:2025-01-31, after:2025-01-01   by the date of the post
//
// A leading "-" negates a word or a has: operator, and a value with spaces
// goes in double quotes: in:"reading list".
type search struct {
	words, notWords []string
	clauses         []string
	args            []any
	gone            bool
}

func parseSearch(input string) search {
	var s search
	for _, tok := range tokenize(input) {
		neg := strings.HasPrefix(tok, "-") && len(tok) > 1
		if neg {
			tok = tok[1:]
		}
		key, value, isOp := strings.Cut(tok, ":")
		value = strings.Trim(value, `"`)
		if !isOp || value == "" || !s.operator(strings.ToLower(key), value, neg) {
			word := strings.Trim(tok, `"`)
			if word == "" {
				continue
			}
			if neg {
				s.notWords = append(s.notWords, word)
			} else {
				s.words = append(s.words, word)
			}
		}
	}
	return s
}

// operator adds the clause for key:value and reports whether key is an
// operator at all.
func (s *search) operator(key, value string, neg bool) bool {
	not := ""
	if neg {
		not = "NOT "
	}
	add := func(clause string, args ...any) {
		s.clauses = append(s.clauses, not+"("+clause+")")
		s.args = append(s.args, args...)
	}
	switch key {
	case "from":
		add("t.author_handle = ? COLLATE NOCASE", strings.TrimPrefix(value, "@"))
	case "has":
		clause, ok := hasClauses[strings.ToLower(value)]
		if !ok {
			return false
		}
		add(clause)
	case "in":
		if strings.EqualFold(value, "none") || strings.EqualFold(value, "any") {
			clause := "EXISTS (SELECT 1 FROM collection_items ci WHERE ci.tweet_id = b.tweet_id)"
			if strings.EqualFold(value, "none") != neg {
				clause = "NOT " + clause
			}
			s.clauses = append(s.clauses, clause)
			return true
		}
		add(`EXISTS (SELECT 1 FROM collection_items ci JOIN collections c ON c.id = ci.collection_id
			WHERE ci.tweet_id = b.tweet_id AND c.name = ? COLLATE NOCASE)`, value)
	case "is":
		switch strings.ToLower(value) {
		case "gone":
			s.gone = !neg
		case "opened":
			add("b.opened_at IS NOT NULL")
		case "unopened":
			add("b.opened_at IS NULL")
		case "aside":
			add("b.aside_at IS NOT NULL")
		default:
			return false
		}
	case "before", "after":
		day, err := time.ParseInLocation("2006-01-02", value, time.Local)
		if err != nil {
			return false
		}
		if key == "before" {
			add("t.created_at < ?", day.UnixMilli())
		} else {
			add("t.created_at >= ?", day.AddDate(0, 0, 1).UnixMilli())
		}
	default:
		return false
	}
	return true
}

func mediaKind(kind string) string {
	return fmt.Sprintf("EXISTS (SELECT 1 FROM json_each(t.data, '$.media') WHERE json_extract(value, '$.kind') = '%s')", kind)
}

var hasClauses = map[string]string{
	"media": "t.has_media",
	"photo": mediaKind("photo"),
	"video": mediaKind("video"),
	"gif":   mediaKind("animated_gif"),
	"link":  "json_array_length(t.data, '$.links') > 0",
	"quote": "json_extract(t.data, '$.quoted') IS NOT NULL",
	"note":  "b.note != ''",
	"article": `EXISTS (SELECT 1 FROM json_each(t.data, '$.links') je JOIN links l ON l.url = json_extract(je.value, '$.expanded')
		WHERE json_extract(l.page, '$.text') IS NOT NULL)`,
	"highlight": "EXISTS (SELECT 1 FROM highlights h WHERE h.tweet_id = b.tweet_id)",
	"thread":    "EXISTS (SELECT 1 FROM threads th WHERE th.tweet_id = b.tweet_id AND th.length > 1)",
}

// tokenize splits on spaces outside double quotes.
func tokenize(input string) []string {
	var out []string
	var cur strings.Builder
	quoted := false
	for _, r := range input {
		switch {
		case r == '"':
			quoted = !quoted
			cur.WriteRune(r)
		case unicode.IsSpace(r) && !quoted:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// ftsTerms turns words into FTS5 prefix terms joined by AND.
func ftsTerms(words []string) string {
	var terms []string
	for _, w := range words {
		w = strings.ReplaceAll(w, `"`, "")
		if w != "" {
			terms = append(terms, `"`+w+`"*`)
		}
	}
	return strings.Join(terms, " ")
}
