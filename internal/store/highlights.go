package store

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"roost/internal/xparse"
)

// Highlight is a passage the user marked in a post or in a page it links to.
type Highlight struct {
	ID      int64  `json:"id"`
	TweetID string `json:"tweetId"`
	// Source is the URL of the linked page the passage is from, or empty
	// for the post itself or its thread.
	Source    string    `json:"source"`
	Quote     string    `json:"quote"`
	CreatedAt time.Time `json:"createdAt"`
	// Tweet is the post the highlight belongs to, filled in by AllHighlights.
	Tweet *xparse.Tweet `json:"tweet,omitempty"`
}

func (s *Store) AddHighlight(tweetID, source, quote string, now time.Time) (Highlight, error) {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return Highlight{}, errors.New("select some text to highlight")
	}
	h := Highlight{TweetID: tweetID, Source: source, Quote: quote, CreatedAt: time.UnixMilli(now.UnixMilli())}
	err := s.db.QueryRow(`INSERT INTO highlights (tweet_id, source, quote, created_at) VALUES (?, ?, ?, ?) RETURNING id`,
		tweetID, source, quote, now.UnixMilli()).Scan(&h.ID)
	return h, err
}

func (s *Store) DeleteHighlight(id int64) error {
	_, err := s.db.Exec(`DELETE FROM highlights WHERE id = ?`, id)
	return err
}

// Highlights returns a post's highlights, oldest first.
func (s *Store) Highlights(tweetID string) ([]Highlight, error) {
	return s.highlights(`WHERE h.tweet_id = ? ORDER BY h.created_at`, false, tweetID)
}

// AllHighlights returns every highlight with its post, newest first.
func (s *Store) AllHighlights() ([]Highlight, error) {
	return s.highlights(`ORDER BY h.created_at DESC`, true)
}

func (s *Store) highlights(tail string, withTweet bool, args ...any) ([]Highlight, error) {
	rows, err := s.db.Query(`SELECT h.id, h.tweet_id, h.source, h.quote, h.created_at, t.data
		FROM highlights h JOIN tweets t ON t.id = h.tweet_id `+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Highlight{}
	for rows.Next() {
		var h Highlight
		var at int64
		var data string
		if err := rows.Scan(&h.ID, &h.TweetID, &h.Source, &h.Quote, &at, &data); err != nil {
			return nil, err
		}
		h.CreatedAt = time.UnixMilli(at)
		if withTweet {
			h.Tweet = &xparse.Tweet{}
			if err := json.Unmarshal([]byte(data), h.Tweet); err != nil {
				return nil, err
			}
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Get returns one bookmark, current or removed.
func (s *Store) Get(tweetID string) (Bookmark, error) {
	bs, err := s.selectBookmarks("b.tweet_id = ?", []any{tweetID}, "")
	if err != nil {
		return Bookmark{}, err
	}
	if len(bs) == 0 {
		return Bookmark{}, errors.New("no bookmark " + tweetID)
	}
	return bs[0], nil
}
