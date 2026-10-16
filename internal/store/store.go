// Package store keeps the bookmark library in SQLite: posts, when each was
// bookmarked and last seen on X, the user's collections and notes, and a log
// of syncs.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"roost/internal/xparse"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

var migrations = []string{`
CREATE TABLE tweets (
	rowid INTEGER PRIMARY KEY,
	id TEXT NOT NULL UNIQUE,
	author_handle TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	has_media INTEGER NOT NULL,
	data TEXT NOT NULL,
	raw TEXT NOT NULL
);
CREATE INDEX tweets_author ON tweets(author_handle COLLATE NOCASE);
CREATE VIRTUAL TABLE tweets_fts USING fts5(text, author, quoted, tokenize = 'unicode61 remove_diacritics 2');

CREATE TABLE bookmarks (
	tweet_id TEXT PRIMARY KEY REFERENCES tweets(id),
	sort_index INTEGER NOT NULL,
	first_seen_at INTEGER NOT NULL,
	last_seen_at INTEGER NOT NULL,
	removed_at INTEGER,
	note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX bookmarks_order ON bookmarks(sort_index DESC);

CREATE TABLE collections (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE COLLATE NOCASE,
	hue INTEGER NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE TABLE collection_items (
	collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
	tweet_id TEXT NOT NULL REFERENCES tweets(id),
	added_at INTEGER NOT NULL,
	PRIMARY KEY (collection_id, tweet_id)
);
CREATE INDEX collection_items_tweet ON collection_items(tweet_id);

CREATE TABLE syncs (
	id INTEGER PRIMARY KEY,
	full INTEGER NOT NULL,
	started_at INTEGER NOT NULL,
	finished_at INTEGER NOT NULL,
	added INTEGER NOT NULL,
	seen INTEGER NOT NULL,
	removed INTEGER NOT NULL,
	reached_end INTEGER NOT NULL,
	error TEXT NOT NULL DEFAULT ''
);
`, `
ALTER TABLE collections ADD COLUMN query TEXT NOT NULL DEFAULT '';
`, `
ALTER TABLE bookmarks ADD COLUMN opened_at INTEGER;
ALTER TABLE bookmarks ADD COLUMN aside_at INTEGER;
`, `
CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`, `
CREATE TABLE threads (
	tweet_id TEXT PRIMARY KEY REFERENCES tweets(id),
	checked_at INTEGER NOT NULL,
	length INTEGER NOT NULL DEFAULT 0,
	posts TEXT NOT NULL DEFAULT '[]'
);
CREATE VIRTUAL TABLE threads_fts USING fts5(text, tokenize = 'unicode61 remove_diacritics 2');
`, `
CREATE TABLE links (
	url TEXT PRIMARY KEY,
	fetched_at INTEGER NOT NULL,
	page TEXT NOT NULL
);
CREATE VIRTUAL TABLE links_fts USING fts5(title, text, tokenize = 'unicode61 remove_diacritics 2');
`, `
CREATE TABLE highlights (
	id INTEGER PRIMARY KEY,
	tweet_id TEXT NOT NULL REFERENCES tweets(id),
	source TEXT NOT NULL DEFAULT '',
	quote TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX highlights_tweet ON highlights (tweet_id);
`, `
CREATE VIRTUAL TABLE notes_fts USING fts5(text, tokenize = 'unicode61 remove_diacritics 2');
INSERT INTO notes_fts (rowid, text)
	SELECT t.rowid, b.note || ' ' || coalesce((SELECT group_concat(h.quote, ' ') FROM highlights h WHERE h.tweet_id = t.id), '')
	FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id;
`, `
ALTER TABLE highlights ADD COLUMN note TEXT NOT NULL DEFAULT '';
`}

// Open opens the library at path, creating and migrating it as needed.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		db.Close()
		return nil, fmt.Errorf("open library: %w", err)
	}
	for i := version; i < len(migrations); i++ {
		if _, err := db.Exec(migrations[i]); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate library to version %d: %w", i+1, err)
		}
		if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			db.Close()
			return nil, err
		}
	}
	s := &Store{db: db}
	if err := s.reparse(); err != nil {
		db.Close()
		return nil, fmt.Errorf("update library: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// parserVersion rises whenever xparse reads a field it did not read before.
// Opening a library written by an older parser parses every post again from
// the raw object it keeps, so posts saved earlier gain the field.
const parserVersion = 1

func (s *Store) reparse() error {
	var version int
	err := s.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM settings WHERE key = 'parser_version'`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if version >= parserVersion {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	type post struct {
		rowid int64
		raw   string
	}
	var posts []post
	rows, err := tx.Query(`SELECT rowid, raw FROM tweets`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p post
		if err := rows.Scan(&p.rowid, &p.raw); err != nil {
			rows.Close()
			return err
		}
		posts = append(posts, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range posts {
		t, ok := xparse.ParseTweet([]byte(p.raw))
		if !ok {
			continue
		}
		data, _ := json.Marshal(t)
		if _, err := tx.Exec(`UPDATE tweets SET data = ? WHERE rowid = ?`, string(data), p.rowid); err != nil {
			return err
		}
		if err := index(tx, p.rowid, t); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES ('parser_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, fmt.Sprint(parserVersion)); err != nil {
		return err
	}
	return tx.Commit()
}

// index makes a post findable by its text and its translation, its author,
// and the post it quotes.
func index(tx *sql.Tx, rowid int64, t xparse.Tweet) error {
	text := t.Text
	if t.Translation != nil {
		text += "\n" + t.Translation.Text
	}
	quoted := ""
	if q := t.Quoted; q != nil {
		quoted = q.AuthorHandle + " " + q.AuthorName + " " + q.Text
		if q.Translation != nil {
			quoted += "\n" + q.Translation.Text
		}
	}
	if _, err := tx.Exec(`DELETE FROM tweets_fts WHERE rowid = ?`, rowid); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT INTO tweets_fts (rowid, text, author, quoted) VALUES (?, ?, ?, ?)`,
		rowid, text, t.AuthorHandle+" "+t.AuthorName, quoted)
	return err
}

// Ingest records entries seen on X at seenAt. An entry is new when the
// library did not hold it as a current bookmark, which includes a post
// bookmarked again after being removed.
func (s *Store) Ingest(entries []xparse.Entry, seenAt time.Time) (added int, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := seenAt.UnixMilli()
	for _, e := range entries {
		t := e.Tweet
		data, _ := json.Marshal(t)
		var rowid int64
		err := tx.QueryRow(`
			INSERT INTO tweets (id, author_handle, created_at, has_media, data, raw) VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET author_handle = excluded.author_handle, has_media = excluded.has_media,
				data = excluded.data, raw = excluded.raw
			RETURNING rowid`,
			t.ID, t.AuthorHandle, t.CreatedAt.UnixMilli(), len(t.Media) > 0, string(data), string(t.Raw)).Scan(&rowid)
		if err != nil {
			return 0, fmt.Errorf("store post %s: %w", t.ID, err)
		}
		if err := index(tx, rowid, t); err != nil {
			return 0, err
		}

		var current bool
		err = tx.QueryRow(`SELECT removed_at IS NULL FROM bookmarks WHERE tweet_id = ?`, t.ID).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			current = false
		} else if err != nil {
			return 0, err
		}
		if !current {
			added++
		}
		if _, err := tx.Exec(`
			INSERT INTO bookmarks (tweet_id, sort_index, first_seen_at, last_seen_at) VALUES (?, ?, ?, ?)
			ON CONFLICT(tweet_id) DO UPDATE SET sort_index = excluded.sort_index,
				last_seen_at = excluded.last_seen_at, removed_at = NULL`,
			t.ID, e.SortIndex, now, now); err != nil {
			return 0, err
		}
	}
	return added, tx.Commit()
}

// MarkRemoved marks every current bookmark not seen since syncStart as
// removed on X. Call it only after a sync has read the whole timeline.
func (s *Store) MarkRemoved(syncStart, now time.Time) (int, error) {
	res, err := s.db.Exec(`UPDATE bookmarks SET removed_at = ? WHERE removed_at IS NULL AND last_seen_at < ?`,
		now.UnixMilli(), syncStart.UnixMilli())
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// Query selects bookmarks for the library view.
type Query struct {
	Search string `json:"search"`
	// Collection is a collection ID, 0 for every bookmark, or Unsorted.
	Collection int64  `json:"collection"`
	Author     string `json:"author"`
	MediaOnly  bool   `json:"mediaOnly"`
	// Removed lists bookmarks removed on X instead of current ones.
	Removed bool `json:"removed"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
}

// Unsorted selects bookmarks in no collection.
const Unsorted = -1

type Bookmark struct {
	Tweet       xparse.Tweet `json:"tweet"`
	Note        string       `json:"note"`
	Collections []int64      `json:"collections"`
	FirstSeenAt time.Time    `json:"firstSeenAt"`
	RemovedAt   *time.Time   `json:"removedAt"`
	OpenedAt    *time.Time   `json:"openedAt"`
	AsideAt     *time.Time   `json:"asideAt"`
	// ThreadLength is the number of posts in the post's thread, or 0 if it
	// stands alone or has not been checked.
	ThreadLength int `json:"threadLength"`
}

type ListResult struct {
	Items []Bookmark `json:"items"`
	Total int        `json:"total"`
}

func (s *Store) List(q Query) (ListResult, error) {
	text := q.Search
	var where []string
	var args []any
	switch {
	case q.Collection == Unsorted:
		where = append(where, "NOT EXISTS (SELECT 1 FROM collection_items ci WHERE ci.tweet_id = b.tweet_id)")
	case q.Collection > 0:
		var smart string
		if err := s.db.QueryRow(`SELECT query FROM collections WHERE id = ?`, q.Collection).Scan(&smart); err != nil {
			return ListResult{}, fmt.Errorf("find collection %d: %w", q.Collection, err)
		}
		if smart != "" {
			text = smart + " " + text
		} else {
			where = append(where, "EXISTS (SELECT 1 FROM collection_items ci WHERE ci.tweet_id = b.tweet_id AND ci.collection_id = ?)")
			args = append(args, q.Collection)
		}
	}
	srch := parseSearch(text)
	if q.Removed || srch.gone {
		where = append(where, "b.removed_at IS NOT NULL")
	} else {
		where = append(where, "b.removed_at IS NULL")
	}
	// Words match the post, the rest of its thread, a page it links to, or
	// the user's own note and highlights.
	const matches = `(t.rowid IN (SELECT rowid FROM tweets_fts WHERE tweets_fts MATCH ?)
		OR t.rowid IN (SELECT rowid FROM threads_fts WHERE threads_fts MATCH ?)
		OR t.rowid IN (SELECT rowid FROM notes_fts WHERE notes_fts MATCH ?)
		OR EXISTS (SELECT 1 FROM json_each(t.data, '$.links') je JOIN links l ON l.url = json_extract(je.value, '$.expanded')
			WHERE l.rowid IN (SELECT rowid FROM links_fts WHERE links_fts MATCH ?)))`
	if m := ftsTerms(srch.words); m != "" {
		where = append(where, matches)
		args = append(args, m, m, m, m)
	}
	for _, w := range srch.notWords {
		m := ftsTerms([]string{w})
		where = append(where, "NOT "+matches)
		args = append(args, m, m, m, m)
	}
	where = append(where, srch.clauses...)
	args = append(args, srch.args...)
	from := "bookmarks b JOIN tweets t ON t.id = b.tweet_id"
	if q.Author != "" {
		where = append(where, "t.author_handle = ? COLLATE NOCASE")
		args = append(args, q.Author)
	}
	if q.MediaOnly {
		where = append(where, "t.has_media")
	}
	cond := strings.Join(where, " AND ")

	var res ListResult
	if err := s.db.QueryRow("SELECT count(*) FROM "+from+" WHERE "+cond, args...).Scan(&res.Total); err != nil {
		return res, fmt.Errorf("count bookmarks: %w", err)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 60
	}
	var err error
	res.Items, err = s.selectBookmarks(cond, args, "ORDER BY b.sort_index DESC LIMIT ? OFFSET ?", limit, q.Offset)
	return res, err
}

// selectBookmarks reads the bookmarks matching cond, a condition over
// bookmarks b and tweets t, followed by tail (ORDER BY and LIMIT).
func (s *Store) selectBookmarks(cond string, args []any, tail string, tailArgs ...any) ([]Bookmark, error) {
	rows, err := s.db.Query(`
		SELECT t.data, b.note, b.first_seen_at, b.removed_at, b.opened_at, b.aside_at,
			(SELECT group_concat(collection_id) FROM collection_items ci WHERE ci.tweet_id = b.tweet_id),
			coalesce((SELECT length FROM threads th WHERE th.tweet_id = b.tweet_id), 0)
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id WHERE `+cond+` `+tail,
		append(append([]any{}, args...), tailArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()
	out := []Bookmark{}
	for rows.Next() {
		var data, note string
		var firstSeen int64
		var removed, opened, aside sql.NullInt64
		var cols sql.NullString
		var threadLength int
		if err := rows.Scan(&data, &note, &firstSeen, &removed, &opened, &aside, &cols, &threadLength); err != nil {
			return nil, err
		}
		b := Bookmark{Note: note, FirstSeenAt: time.UnixMilli(firstSeen), Collections: []int64{},
			RemovedAt: optTime(removed), OpenedAt: optTime(opened), AsideAt: optTime(aside), ThreadLength: threadLength}
		if err := json.Unmarshal([]byte(data), &b.Tweet); err != nil {
			return nil, err
		}
		for _, c := range strings.Split(cols.String, ",") {
			var id int64
			if _, err := fmt.Sscan(c, &id); err == nil {
				b.Collections = append(b.Collections, id)
			}
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func optTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.UnixMilli(v.Int64)
	return &t
}

// MarkOpened records that the user opened the bookmark in roost.
func (s *Store) MarkOpened(tweetID string, now time.Time) error {
	_, err := s.db.Exec(`UPDATE bookmarks SET opened_at = ? WHERE tweet_id = ?`, now.UnixMilli(), tweetID)
	return err
}

// SetAside takes a bookmark out of the sorting queue, or puts it back,
// without filing it.
func (s *Store) SetAside(tweetID string, aside bool, now time.Time) error {
	var at any
	if aside {
		at = now.UnixMilli()
	}
	_, err := s.db.Exec(`UPDATE bookmarks SET aside_at = ? WHERE tweet_id = ?`, at, tweetID)
	return err
}

// Revisit is a handful of bookmarks worth seeing again.
type Revisit struct {
	// OnThisDay holds posts written on today's date in earlier years.
	OnThisDay []Bookmark `json:"onThisDay"`
	// Unopened holds bookmarks never opened in roost, at random.
	Unopened []Bookmark `json:"unopened"`
	// Random holds any current bookmarks, at random.
	Random []Bookmark `json:"random"`
}

func (s *Store) Revisit(now time.Time, n int) (Revisit, error) {
	var r Revisit
	var err error
	day := now.Format("01-02")
	r.OnThisDay, err = s.selectBookmarks(
		`b.removed_at IS NULL AND strftime('%m-%d', t.created_at / 1000, 'unixepoch', 'localtime') = ?
			AND strftime('%Y', t.created_at / 1000, 'unixepoch', 'localtime') < ?`,
		[]any{day, now.Format("2006")}, "ORDER BY t.created_at DESC LIMIT ?", n)
	if err != nil {
		return r, err
	}
	r.Unopened, err = s.selectBookmarks("b.removed_at IS NULL AND b.opened_at IS NULL", nil, "ORDER BY random() LIMIT ?", n)
	if err != nil {
		return r, err
	}
	r.Random, err = s.selectBookmarks("b.removed_at IS NULL", nil, "ORDER BY random() LIMIT ?", n)
	return r, err
}

type Collection struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Hue  int    `json:"hue"`
	// Query makes a smart collection: it holds the bookmarks matching this
	// search instead of the ones filed into it.
	Query string `json:"query"`
	Count int    `json:"count"`
}

func (s *Store) Collections() ([]Collection, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.hue, c.query,
			(SELECT count(*) FROM collection_items ci JOIN bookmarks b ON b.tweet_id = ci.tweet_id
			 WHERE ci.collection_id = c.id AND b.removed_at IS NULL)
		FROM collections c ORDER BY c.query != '', c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.Hue, &c.Query, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i, c := range out {
		if c.Query != "" {
			res, err := s.List(Query{Collection: c.ID, Limit: 1})
			if err != nil {
				return nil, err
			}
			out[i].Count = res.Total
		}
	}
	return out, nil
}

// CreateCollection makes a collection; a non-empty query makes it smart.
func (s *Store) CreateCollection(name string, hue int, query string, now time.Time) (Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Collection{}, errors.New("a collection needs a name")
	}
	c := Collection{Name: name, Hue: hue, Query: strings.TrimSpace(query)}
	err := s.db.QueryRow(`INSERT INTO collections (name, hue, query, created_at) VALUES (?, ?, ?, ?) RETURNING id`,
		name, hue, c.Query, now.UnixMilli()).Scan(&c.ID)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return c, fmt.Errorf("a collection named %q already exists", name)
	}
	return c, err
}

func (s *Store) UpdateCollection(id int64, name string, hue int, query string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("a collection needs a name")
	}
	_, err := s.db.Exec(`UPDATE collections SET name = ?, hue = ?, query = ? WHERE id = ?`, name, hue, strings.TrimSpace(query), id)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("a collection named %q already exists", name)
	}
	return err
}

func (s *Store) DeleteCollection(id int64) error {
	_, err := s.db.Exec(`DELETE FROM collections WHERE id = ?`, id)
	return err
}

// SetInCollection adds the post to the collection or takes it out.
func (s *Store) SetInCollection(tweetID string, collectionID int64, in bool, now time.Time) error {
	var smart string
	if err := s.db.QueryRow(`SELECT query FROM collections WHERE id = ?`, collectionID).Scan(&smart); err != nil {
		return fmt.Errorf("find collection %d: %w", collectionID, err)
	}
	if smart != "" {
		return errors.New("a smart collection holds whatever matches its search; posts can't be filed into it")
	}
	var err error
	if in {
		_, err = s.db.Exec(`INSERT OR IGNORE INTO collection_items (collection_id, tweet_id, added_at) VALUES (?, ?, ?)`,
			collectionID, tweetID, now.UnixMilli())
	} else {
		_, err = s.db.Exec(`DELETE FROM collection_items WHERE collection_id = ? AND tweet_id = ?`, collectionID, tweetID)
	}
	return err
}

func (s *Store) SetNote(tweetID, note string) error {
	if _, err := s.db.Exec(`UPDATE bookmarks SET note = ? WHERE tweet_id = ?`, strings.TrimSpace(note), tweetID); err != nil {
		return err
	}
	return s.indexNotes(tweetID)
}

// indexNotes makes the words the user kept about a post searchable: its
// note, its highlights and their notes.
func (s *Store) indexNotes(tweetID string) error {
	if _, err := s.db.Exec(`DELETE FROM notes_fts WHERE rowid = (SELECT rowid FROM tweets WHERE id = ?)`, tweetID); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO notes_fts (rowid, text)
		SELECT t.rowid, b.note || ' ' || coalesce((SELECT group_concat(h.quote || ' ' || h.note, ' ') FROM highlights h WHERE h.tweet_id = t.id), '')
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id WHERE t.id = ?`, tweetID)
	return err
}

type Author struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Count  int    `json:"count"`
}

// Authors returns the authors with the most current bookmarks.
func (s *Store) Authors(limit int) ([]Author, error) {
	rows, err := s.db.Query(`
		SELECT t.author_handle, json_extract(max(t.data), '$.authorName'), json_extract(max(t.data), '$.authorAvatar'), count(*) AS n
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id
		WHERE b.removed_at IS NULL
		GROUP BY t.author_handle COLLATE NOCASE ORDER BY n DESC, t.author_handle LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Author{}
	for rows.Next() {
		var a Author
		var name, avatar sql.NullString
		if err := rows.Scan(&a.Handle, &name, &avatar, &a.Count); err != nil {
			return nil, err
		}
		a.Name, a.Avatar = name.String, avatar.String
		out = append(out, a)
	}
	return out, rows.Err()
}

type Sync struct {
	Full       bool      `json:"full"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Added      int       `json:"added"`
	Seen       int       `json:"seen"`
	Removed    int       `json:"removed"`
	ReachedEnd bool      `json:"reachedEnd"`
	Error      string    `json:"error"`
}

func (s *Store) RecordSync(r Sync) error {
	_, err := s.db.Exec(`INSERT INTO syncs (full, started_at, finished_at, added, seen, removed, reached_end, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, r.Full, r.StartedAt.UnixMilli(), r.FinishedAt.UnixMilli(),
		r.Added, r.Seen, r.Removed, r.ReachedEnd, r.Error)
	return err
}

// LastFullSync is when a sync last read the whole timeline without error,
// or the zero time if none has.
func (s *Store) LastFullSync() (time.Time, error) {
	var at sql.NullInt64
	err := s.db.QueryRow(`SELECT max(finished_at) FROM syncs WHERE full AND reached_end AND error = ''`).Scan(&at)
	if err != nil || !at.Valid {
		return time.Time{}, err
	}
	return time.UnixMilli(at.Int64), nil
}

type Stats struct {
	Total    int `json:"total"`
	Removed  int `json:"removed"`
	Unsorted int `json:"unsorted"`
	// ToSort counts unsorted bookmarks not set aside: the sorting queue.
	ToSort   int   `json:"toSort"`
	LastSync *Sync `json:"lastSync"`
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	err := s.db.QueryRow(`
		SELECT count(*) FILTER (WHERE removed_at IS NULL), count(*) FILTER (WHERE removed_at IS NOT NULL),
			count(*) FILTER (WHERE removed_at IS NULL AND NOT EXISTS (SELECT 1 FROM collection_items ci WHERE ci.tweet_id = bookmarks.tweet_id)),
			count(*) FILTER (WHERE removed_at IS NULL AND aside_at IS NULL AND NOT EXISTS (SELECT 1 FROM collection_items ci WHERE ci.tweet_id = bookmarks.tweet_id))
		FROM bookmarks`).Scan(&st.Total, &st.Removed, &st.Unsorted, &st.ToSort)
	if err != nil {
		return st, err
	}
	var r Sync
	var started, finished int64
	err = s.db.QueryRow(`SELECT full, started_at, finished_at, added, seen, removed, reached_end, error
		FROM syncs ORDER BY id DESC LIMIT 1`).Scan(&r.Full, &started, &finished, &r.Added, &r.Seen, &r.Removed, &r.ReachedEnd, &r.Error)
	if errors.Is(err, sql.ErrNoRows) {
		return st, nil
	}
	r.StartedAt, r.FinishedAt = time.UnixMilli(started), time.UnixMilli(finished)
	st.LastSync = &r
	return st, err
}
