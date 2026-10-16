package store

import (
	"database/sql"
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
	Source string `json:"source"`
	Quote  string `json:"quote"`
	// Note is what the user wrote about the passage.
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
	// Tweet is the post the highlight belongs to, filled in by AllHighlights.
	Tweet *xparse.Tweet `json:"tweet,omitempty"`
}

func (s *Store) AddHighlight(tweetID, source, quote string, now time.Time) (Highlight, error) {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return Highlight{}, errors.New("select some text to highlight")
	}
	if len([]rune(quote)) > maxHighlightQuote {
		return Highlight{}, errors.New("highlight is too long")
	}
	h := Highlight{TweetID: tweetID, Source: source, Quote: quote, CreatedAt: time.UnixMilli(now.UnixMilli())}
	tx, err := s.db.Begin()
	if err != nil {
		return h, err
	}
	defer tx.Rollback()
	err = tx.QueryRow(`INSERT INTO highlights (tweet_id, source, quote, created_at) VALUES (?, ?, ?, ?) RETURNING id`,
		tweetID, source, quote, now.UnixMilli()).Scan(&h.ID)
	if err != nil {
		return h, err
	}
	if err := indexNotes(tx, tweetID); err != nil {
		return h, err
	}
	return h, tx.Commit()
}

// RestoreHighlight puts a deleted highlight back with its original identity and content.
func (s *Store) RestoreHighlight(h Highlight) error {
	h.Quote = strings.TrimSpace(h.Quote)
	h.Note = strings.TrimSpace(h.Note)
	if h.ID <= 0 || h.TweetID == "" || h.Quote == "" || h.CreatedAt.IsZero() {
		return errors.New("that highlight cannot be restored")
	}
	if len([]rune(h.Quote)) > maxHighlightQuote || len([]rune(h.Note)) > maxHighlightNote {
		return errors.New("that highlight is too long")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO highlights (id, tweet_id, source, quote, note, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		h.ID, h.TweetID, h.Source, h.Quote, h.Note, h.CreatedAt.UnixMilli()); err != nil {
		return err
	}
	if err := indexNotes(tx, h.TweetID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteHighlight(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var tweetID string
	err = tx.QueryRow(`DELETE FROM highlights WHERE id = ? RETURNING tweet_id`, id).Scan(&tweetID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := indexNotes(tx, tweetID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetHighlightNote(id int64, note string) error {
	if len([]rune(note)) > maxHighlightNote {
		return errors.New("highlight note is too long")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var tweetID string
	err = tx.QueryRow(`UPDATE highlights SET note = ? WHERE id = ? RETURNING tweet_id`, strings.TrimSpace(note), id).Scan(&tweetID)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("that highlight no longer exists")
	}
	if err != nil {
		return err
	}
	if err := indexNotes(tx, tweetID); err != nil {
		return err
	}
	return tx.Commit()
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
	rows, err := s.db.Query(`SELECT h.id, h.tweet_id, h.source, h.quote, h.note, h.created_at, t.data
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
		if err := rows.Scan(&h.ID, &h.TweetID, &h.Source, &h.Quote, &h.Note, &at, &data); err != nil {
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

// All returns every bookmark, removed ones included, newest bookmark first.
func (s *Store) All() ([]Bookmark, error) {
	return s.selectBookmarks("1 = 1", nil, "ORDER BY b.sort_index DESC")
}
