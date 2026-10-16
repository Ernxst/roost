package store

import (
	"encoding/json"
	"time"

	"roost/internal/links"
	"roost/internal/xparse"
)

// LinkCandidates lists links in current bookmarks that have not been
// fetched yet, newest bookmark first.
func (s *Store) LinkCandidates(limit int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT json_extract(je.value, '$.expanded')
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id, json_each(t.data, '$.links') je
		WHERE b.removed_at IS NULL
			-- json_each yields one NULL row for a post whose links are null.
			AND json_extract(je.value, '$.expanded') IS NOT NULL
			AND json_extract(je.value, '$.expanded') NOT IN (SELECT url FROM links)
		ORDER BY b.sort_index DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() && len(out) < limit {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		if !links.Skip(u) {
			out = append(out, u)
		}
	}
	return out, rows.Err()
}

func (s *Store) SaveLink(p links.Page, now time.Time) error {
	if len([]rune(p.Title)) > 300 {
		p.Title = string([]rune(p.Title)[:300])
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var rowid int64
	if err := tx.QueryRow(`INSERT INTO links (url, fetched_at, page) VALUES (?, ?, ?)
		ON CONFLICT (url) DO UPDATE SET fetched_at = excluded.fetched_at, page = excluded.page
		RETURNING rowid`, p.URL, now.UnixMilli(), string(data)).Scan(&rowid); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM links_fts WHERE rowid = ?`, rowid); err != nil {
		return err
	}
	if p.Title != "" || p.Text != "" {
		if _, err := tx.Exec(`INSERT INTO links_fts (rowid, title, text) VALUES (?, ?, ?)`, rowid, p.Title, p.Text); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SavedLink is a link in a post with what roost saved of the page, without
// its text.
type SavedLink struct {
	xparse.Link
	Page    *links.Page `json:"page"`
	HasText bool        `json:"hasText"`
}

// Links returns the links in a post and the pages saved for them.
func (s *Store) Links(tweetID string) ([]SavedLink, error) {
	var data string
	if err := s.db.QueryRow(`SELECT data FROM tweets WHERE id = ?`, tweetID).Scan(&data); err != nil {
		return nil, err
	}
	var t xparse.Tweet
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, err
	}
	out := []SavedLink{}
	for _, l := range t.Links {
		sl := SavedLink{Link: l}
		if p, err := s.Page(l.Expanded); err == nil && p != nil {
			sl.HasText = p.Text != ""
			p.Text = ""
			sl.Page = p
		}
		out = append(out, sl)
	}
	return out, nil
}

// Page returns the saved page for url, or nil if there is none.
func (s *Store) Page(url string) (*links.Page, error) {
	var data string
	if err := s.db.QueryRow(`SELECT page FROM links WHERE url = ?`, url).Scan(&data); err != nil {
		return nil, nil
	}
	var p links.Page
	return &p, json.Unmarshal([]byte(data), &p)
}
