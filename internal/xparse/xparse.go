// Package xparse turns the bodies of X's Bookmarks GraphQL responses into
// bookmark entries. X moves fields between releases, so the parser walks the
// whole response for timeline entries and reads each field from every place X
// has been seen to put it.
package xparse

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Entry is one bookmark as the timeline lists it.
type Entry struct {
	// SortIndex orders the timeline: a larger value was bookmarked later.
	SortIndex int64
	Tweet     Tweet
}

// Tweet is the part of a post the app shows and searches.
type Tweet struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	AuthorHandle string    `json:"authorHandle"`
	AuthorName   string    `json:"authorName"`
	AuthorAvatar string    `json:"authorAvatar"`
	Text         string    `json:"text"`
	CreatedAt    time.Time `json:"createdAt"`
	Media        []Media   `json:"media"`
	Links        []Link    `json:"links"`
	Quoted       *Tweet    `json:"quoted,omitempty"`
	// ReplyTo is the ID of the post this one replies to.
	ReplyTo string `json:"replyTo,omitempty"`
	// Translation is X's translation of Text into the reader's language,
	// which X sends with posts written in another.
	Translation *Translation `json:"translation,omitempty"`
	// Raw is X's own object for the post, kept so later versions of the
	// parser can read fields this one ignores.
	Raw json.RawMessage `json:"-"`
}

// Translation is a post's text in another language. Languages are BCP 47
// codes such as "ja".
type Translation struct {
	Text string `json:"text"`
	From string `json:"from"`
	To   string `json:"to"`
}

// Media is a photo, video or GIF attached to a post.
type Media struct {
	Kind   string `json:"kind"` // photo, video or animated_gif
	Thumb  string `json:"thumb"`
	Video  string `json:"video,omitempty"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Link is a URL in the text, which X shortens to t.co.
type Link struct {
	Short    string `json:"short"`
	Expanded string `json:"expanded"`
	Display  string `json:"display"`
}

// Parse returns the bookmark entries in one response body, in timeline
// order. An empty result means the timeline has no more entries.
func Parse(body []byte) ([]Entry, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("decode bookmarks response: %w", err)
	}
	var entries []Entry
	var walk func(any)
	walk = func(node any) {
		switch n := node.(type) {
		case []any:
			for _, v := range n {
				walk(v)
			}
		case map[string]any:
			if id, _ := n["entryId"].(string); strings.HasPrefix(id, "tweet-") {
				result := dig(n, "content", "itemContent", "tweet_results", "result")
				if t, ok := parseTweet(result); ok {
					idx, _ := strconv.ParseInt(str(n["sortIndex"]), 10, 64)
					entries = append(entries, Entry{SortIndex: idx, Tweet: t})
				}
				return
			}
			for _, v := range n {
				walk(v)
			}
		}
	}
	walk(root)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].SortIndex > entries[j].SortIndex })
	return entries, nil
}

func parseTweet(node any) (Tweet, bool) {
	r, _ := node.(map[string]any)
	if r["__typename"] == "TweetWithVisibilityResults" {
		r, _ = r["tweet"].(map[string]any)
	}
	legacy, _ := r["legacy"].(map[string]any)
	id := str(r["rest_id"])
	if legacy == nil || id == "" {
		return Tweet{}, false
	}
	user, _ := dig(r, "core", "user_results", "result").(map[string]any)
	handle := first(str(dig(user, "core", "screen_name")), str(dig(user, "legacy", "screen_name")))
	t := Tweet{
		ID:           id,
		URL:          "https://x.com/" + handle + "/status/" + id,
		AuthorHandle: handle,
		AuthorName:   first(str(dig(user, "core", "name")), str(dig(user, "legacy", "name"))),
		AuthorAvatar: first(str(dig(user, "avatar", "image_url")), str(dig(user, "legacy", "profile_image_url_https"))),
		Text:         str(legacy["full_text"]),
		ReplyTo:      str(legacy["in_reply_to_status_id_str"]),
	}
	t.CreatedAt, _ = time.Parse(time.RubyDate, str(legacy["created_at"]))

	urls := dig(legacy, "entities", "urls")
	if note, _ := dig(r, "note_tweet", "note_tweet_results", "result").(map[string]any); note != nil {
		t.Text = str(note["text"])
		urls = dig(note, "entity_set", "urls")
	}
	for _, u := range list(urls) {
		t.Links = append(t.Links, Link{Short: str(dig(u, "url")), Expanded: str(dig(u, "expanded_url")), Display: str(dig(u, "display_url"))})
	}

	media := list(dig(legacy, "extended_entities", "media"))
	for _, m := range media {
		item := Media{
			Kind:   str(dig(m, "type")),
			Thumb:  str(dig(m, "media_url_https")),
			Width:  num(dig(m, "original_info", "width")),
			Height: num(dig(m, "original_info", "height")),
		}
		best := -1
		for _, v := range list(dig(m, "video_info", "variants")) {
			if str(dig(v, "content_type")) == "video/mp4" && num(dig(v, "bitrate")) > best {
				best, item.Video = num(dig(v, "bitrate")), str(dig(v, "url"))
			}
		}
		t.Media = append(t.Media, item)
	}
	if tr := dig(r, "grok_translated_post_with_availability"); dig(tr, "is_available") == true {
		from, to := str(dig(tr, "data", "source_language")), str(dig(tr, "data", "destination_language"))
		if text := str(dig(tr, "data", "translation")); text != "" && from != to {
			t.Translation = &Translation{Text: text, From: from, To: to}
		}
	}
	// X appends a t.co link for attached media to the text; it duplicates
	// the media shown below the text.
	if len(media) > 0 {
		if short := str(dig(media[0], "url")); short != "" {
			t.Text = strings.TrimSpace(strings.TrimSuffix(t.Text, short))
			if t.Translation != nil {
				t.Translation.Text = strings.TrimSpace(strings.TrimSuffix(t.Translation.Text, short))
			}
		}
	}

	if q, ok := parseTweet(dig(r, "quoted_status_result", "result")); ok {
		t.Quoted = &q
	}
	t.Raw, _ = json.Marshal(r)
	return t, true
}

// ParseTweet reads a post from X's own object for it, as Tweet.Raw keeps.
func ParseTweet(raw []byte) (Tweet, bool) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return Tweet{}, false
	}
	return parseTweet(node)
}

func dig(node any, path ...string) any {
	for _, key := range path {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[key]
	}
	return node
}

func list(node any) []any { l, _ := node.([]any); return l }

func str(node any) string { s, _ := node.(string); return s }

func num(node any) int { f, _ := node.(float64); return int(f) }

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
