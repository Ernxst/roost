package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"roost/internal/xparse"
)

// Settings are the user's preferences.
type Settings struct {
	// CacheVideos keeps videos on disk as well as images.
	CacheVideos bool `json:"cacheVideos"`
	// SyncEvery is the minutes between background syncs; 0 turns them off.
	SyncEvery int `json:"syncEvery"`
	// Notify announces the bookmarks a background sync found or found gone.
	Notify bool `json:"notify"`
	// ExportDir is where the last Markdown export went.
	ExportDir string `json:"exportDir"`
}

var defaultSettings = Settings{SyncEvery: 60, Notify: true}

func (s *Store) Settings() (Settings, error) {
	out := defaultSettings
	var raw string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'settings'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal([]byte(raw), &out)
}

func (s *Store) SaveSettings(v Settings) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO settings (key, value) VALUES ('settings', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, string(raw))
	return err
}

// MediaURLs lists the media the library shows, as the page requests them:
// avatars, small image thumbnails and video posters, and the videos
// themselves if videos is set.
func (s *Store) MediaURLs(videos bool) ([]string, error) {
	rows, err := s.db.Query(`SELECT t.data FROM tweets t JOIN bookmarks b ON b.tweet_id = t.id ORDER BY b.sort_index DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	var out []string
	add := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	var walk func(t *xparse.Tweet)
	walk = func(t *xparse.Tweet) {
		add(t.AuthorAvatar)
		for _, m := range t.Media {
			add(m.Thumb + "?name=small")
			if m.Video != "" {
				add(m.Thumb)
				if videos {
					add(m.Video)
				}
			}
		}
		if t.Quoted != nil {
			walk(t.Quoted)
		}
	}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var t xparse.Tweet
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, err
		}
		walk(&t)
	}
	return out, rows.Err()
}

// NewAuthors returns the handles of the authors of bookmarks first seen at
// or after since, most frequent first.
func (s *Store) NewAuthors(since time.Time) ([]string, error) {
	rows, err := s.db.Query(`SELECT t.author_handle FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id
		WHERE b.first_seen_at >= ? GROUP BY t.author_handle COLLATE NOCASE
		ORDER BY count(*) DESC, max(b.sort_index) DESC`, since.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
