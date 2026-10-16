package store

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"roost/internal/meaning"
	"roost/internal/xparse"
)

// Suggestion is a collection a post might belong in.
type Suggestion struct {
	CollectionID int64   `json:"collectionId"`
	Score        float64 `json:"score"`
}

// Suggest ranks the manual collections a post is not in by how much the
// post resembles what is already filed there: the words of each post and
// the page it links to, weighted by TF-IDF and compared by cosine
// similarity, plus how often the same author was filed in each collection.
// Where the posts have sentence vectors, how close the post's meaning lies
// to each collection's, against the average collection, counts too.
func (s *Store) Suggest(tweetID string, n int) ([]Suggestion, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.author_handle, t.data, `+linkTitles+`,
			coalesce((SELECT group_concat(ci.collection_id) FROM collection_items ci JOIN collections c ON c.id = ci.collection_id
				WHERE ci.tweet_id = t.id AND c.query = ''), ''),
			v.vector
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id LEFT JOIN vectors v ON v.tweet_id = t.id
		WHERE b.removed_at IS NULL OR t.id = ?`, tweetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type doc struct {
		author string
		terms  map[string]float64
		in     []int64
		vector []float32
	}
	var docs []doc
	var target *doc
	df := map[string]int{}
	for rows.Next() {
		var id, author, data, titles, cols string
		var vector []byte
		if err := rows.Scan(&id, &author, &data, &titles, &cols, &vector); err != nil {
			return nil, err
		}
		var t xparse.Tweet
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			return nil, err
		}
		d := doc{author: strings.ToLower(author), terms: termCounts(suggestText(t, titles)), vector: decodeVector(vector)}
		for _, c := range strings.Split(cols, ",") {
			var cid int64
			if _, err := fmt.Sscan(c, &cid); err == nil {
				d.in = append(d.in, cid)
			}
		}
		for term := range d.terms {
			df[term]++
		}
		docs = append(docs, d)
		if id == tweetID {
			target = &docs[len(docs)-1]
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if target == nil {
		return []Suggestion{}, nil
	}
	targetIn := map[int64]bool{}
	for _, c := range target.in {
		targetIn[c] = true
	}

	total := float64(len(docs))
	weigh := func(terms map[string]float64) map[string]float64 {
		out := make(map[string]float64, len(terms))
		var norm float64
		for term, tf := range terms {
			w := (1 + math.Log(tf)) * math.Log(1+total/float64(df[term]))
			out[term] = w
			norm += w * w
		}
		norm = math.Sqrt(norm)
		for term := range out {
			out[term] /= norm
		}
		return out
	}

	// Each collection's centroid is the sum of its posts' unit vectors, of
	// terms and of meaning.
	centroid := map[int64]map[string]float64{}
	meant := map[int64][]float32{}
	byAuthor := map[int64]float64{}
	authorPosts := 0.0
	for i := range docs {
		d := &docs[i]
		if d == target || len(d.in) == 0 {
			continue
		}
		if d.author == target.author {
			authorPosts++
		}
		v := weigh(d.terms)
		for _, c := range d.in {
			if centroid[c] == nil {
				centroid[c] = map[string]float64{}
			}
			for term, w := range v {
				centroid[c][term] += w
			}
			if d.author == target.author {
				byAuthor[c]++
			}
			if d.vector != nil && len(d.vector) == len(target.vector) {
				meant[c] = addUnit(meant[c], d.vector)
			}
		}
	}

	// Meaning shifts each collection's score by how much closer the post
	// lies to it than to the average collection. Sentence vectors of
	// unrelated posts are still fairly similar, so the raw similarity says
	// little on its own.
	closeness := map[int64]float64{}
	if len(meant) > 0 {
		var mean float64
		for c, v := range meant {
			closeness[c] = meaning.Cosine(v, target.vector)
			mean += closeness[c]
		}
		mean /= float64(len(meant))
		for c := range closeness {
			closeness[c] -= mean
		}
	}
	tv := weigh(target.terms)
	var out []Suggestion
	for c, cv := range centroid {
		if targetIn[c] {
			continue
		}
		var dot, norm float64
		for term, w := range cv {
			norm += w * w
			dot += w * tv[term]
		}
		score := 0.0
		if norm > 0 {
			score = dot / math.Sqrt(norm)
		}
		if authorPosts > 0 {
			score += 0.5 * byAuthor[c] / authorPosts
		}
		score += meaningWeight * closeness[c]
		if score >= 0.08 {
			out = append(out, Suggestion{CollectionID: c, Score: score})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > n {
		out = out[:n]
	}
	if out == nil {
		out = []Suggestion{}
	}
	return out, nil
}

// meaningWeight is how much closeness in meaning counts against shared
// words. It was chosen by leaving each of 333 filed posts in one library out
// in turn: with it, the post's own collection came first for 179 posts
// rather than 159, and among the three suggestions for 226 rather than 187.
// Weights from 0.25 to 0.75 did about as well.
const meaningWeight = 0.5

// linkTitles selects the titles of the saved pages post t links to.
const linkTitles = `coalesce((SELECT group_concat(json_extract(l.page, '$.title'), ' ')
	FROM json_each(t.data, '$.links') je JOIN links l ON l.url = json_extract(je.value, '$.expanded')), '')`

// suggestText is what a post is about, for suggestions: its text, the post
// it quotes and the titles of the pages it links to.
func suggestText(t xparse.Tweet, titles string) string {
	text := t.Text + " " + titles
	if t.Quoted != nil {
		text += " " + t.Quoted.Text
	}
	return text
}

// addUnit adds v, scaled to unit length, to sum.
func addUnit(sum, v []float32) []float32 {
	if sum == nil {
		sum = make([]float32, len(v))
	}
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm == 0 {
		return sum
	}
	scale := float32(1 / math.Sqrt(norm))
	for i, x := range v {
		sum[i] += x * scale
	}
	return sum
}

// PostText is a post to embed and the text that stands for it.
type PostText struct {
	ID   string
	Text string
}

// VectorCandidates lists current bookmarks with no sentence vector from
// model, newest bookmark first, with the text to embed for each.
func (s *Store) VectorCandidates(model string, limit int) ([]PostText, error) {
	rows, err := s.db.Query(`
		SELECT t.data, `+linkTitles+`
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id LEFT JOIN vectors v ON v.tweet_id = t.id
		WHERE b.removed_at IS NULL AND (v.model IS NULL OR v.model != ?)
		ORDER BY b.sort_index DESC LIMIT ?`, model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostText
	for rows.Next() {
		var data, titles string
		if err := rows.Scan(&data, &titles); err != nil {
			return nil, err
		}
		var t xparse.Tweet
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			return nil, err
		}
		out = append(out, PostText{ID: t.ID, Text: urlPattern.ReplaceAllString(suggestText(t, titles), "")})
	}
	return out, rows.Err()
}

var urlPattern = regexp.MustCompile(`https?://\S+`)

// SaveVectors keeps posts' sentence vectors from model, keyed by post ID.
// An empty vector records that the model has none for the post, so it isn't
// asked again. Vectors from any other model are dropped, since they can't be
// compared with these.
func (s *Store) SaveVectors(model string, vectors map[string][]float32) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM vectors WHERE model != ?`, model); err != nil {
		return err
	}
	for id, v := range vectors {
		blob := make([]byte, 4*len(v))
		for i, x := range v {
			binary.LittleEndian.PutUint32(blob[4*i:], math.Float32bits(x))
		}
		if _, err := tx.Exec(`INSERT INTO vectors (tweet_id, model, vector) VALUES (?, ?, ?)
			ON CONFLICT (tweet_id) DO UPDATE SET model = excluded.model, vector = excluded.vector`, id, model, blob); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func decodeVector(blob []byte) []float32 {
	if len(blob) == 0 {
		return nil
	}
	v := make([]float32, len(blob)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[4*i:]))
	}
	return v
}

// termCounts splits text into lower-case words of three letters or more,
// leaving out links, mentions and common words.
func termCounts(text string) map[string]float64 {
	out := map[string]float64{}
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '@' && r != '/' && r != '.' && r != ':'
	}) {
		if strings.Contains(w, "http") || strings.HasPrefix(w, "@") || strings.Contains(w, "/") {
			continue
		}
		w = strings.Trim(w, ".:")
		if len([]rune(w)) < 3 || stopWords[w] {
			continue
		}
		out[w]++
	}
	return out
}

var stopWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`the and for are but not you all any can had her was one our out has have
		this that with from they will your what when there their been were which into more some than then them
		these would about just like also only over such very here how its it's i'm don't can't dont cant get got
		make made really much many most even want need know think going thing things time way well use used using`) {
		stopWords[w] = true
	}
}
