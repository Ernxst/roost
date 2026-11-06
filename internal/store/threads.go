package store

import (
	"encoding/json"
	"strings"
	"time"

	"roost/internal/xparse"
)

// DetailCandidate is a bookmark roost still needs to open on X on its own,
// to check its thread or to save the article it publishes.
type DetailCandidate struct {
	ID  string
	URL string
	// Article is the address of the post's article if roost has not saved
	// it yet.
	Article string
}

// DetailCandidates lists bookmarks to open on X, newest first. They are
// unchecked posts that look like part of a thread and posts with an
// unsaved article. A post looks like part of a thread when it replies to the
// author's own post or announces a thread with 🧵, "thread", "1/" or a
// pointing-down arrow.
func (s *Store) DetailCandidates(limit int) ([]DetailCandidate, error) {
	rows, err := s.db.Query(`
		SELECT id, url, coalesce(article, '') FROM (
			SELECT t.id, json_extract(t.data, '$.url') AS url, b.sort_index,
				(SELECT a.value FROM json_each(t.data, '$.article.url') a
					WHERE a.value NOT IN (SELECT url FROM links)) AS article,
				NOT EXISTS (SELECT 1 FROM threads th WHERE th.tweet_id = t.id)
					AND (json_extract(t.raw, '$.legacy.in_reply_to_user_id_str') = json_extract(t.raw, '$.legacy.user_id_str')
						OR lower(json_extract(t.data, '$.text')) GLOB '*[🧵👇⬇↓]*'
						OR lower(json_extract(t.data, '$.text')) LIKE '%thread%'
						OR json_extract(t.data, '$.text') LIKE '%1/%') AS thread
			FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id
			WHERE b.removed_at IS NULL)
		WHERE thread OR article IS NOT NULL
		ORDER BY sort_index DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DetailCandidate
	for rows.Next() {
		var c DetailCandidate
		if err := rows.Scan(&c.ID, &c.URL, &c.Article); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SaveThread records the thread of a bookmarked post; nil posts records
// that the post stands alone, so it is not checked again.
func (s *Store) SaveThread(tweetID string, posts []xparse.Tweet, now time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if posts == nil {
		posts = []xparse.Tweet{}
	}
	data, err := json.Marshal(posts)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO threads (tweet_id, checked_at, length, posts) VALUES (?, ?, ?, ?)
		ON CONFLICT (tweet_id) DO UPDATE SET checked_at = excluded.checked_at, length = excluded.length, posts = excluded.posts`,
		tweetID, now.UnixMilli(), len(posts), string(data)); err != nil {
		return err
	}
	var rowid int64
	if err := tx.QueryRow(`SELECT rowid FROM tweets WHERE id = ?`, tweetID).Scan(&rowid); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM threads_fts WHERE rowid = ?`, rowid); err != nil {
		return err
	}
	var text []string
	for _, p := range posts {
		if p.ID != tweetID {
			text = append(text, p.Text)
		}
	}
	if len(text) > 0 {
		if _, err := tx.Exec(`INSERT INTO threads_fts (rowid, text) VALUES (?, ?)`, rowid, strings.Join(text, "\n\n")); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Thread returns the posts in a bookmark's thread, or none if it stands
// alone or has not been checked.
func (s *Store) Thread(tweetID string) ([]xparse.Tweet, error) {
	var data string
	err := s.db.QueryRow(`SELECT posts FROM threads WHERE tweet_id = ?`, tweetID).Scan(&data)
	if err != nil {
		return []xparse.Tweet{}, nil
	}
	var posts []xparse.Tweet
	return posts, json.Unmarshal([]byte(data), &posts)
}
