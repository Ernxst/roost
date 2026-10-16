package store

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

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
func (s *Store) Suggest(tweetID string, n int) ([]Suggestion, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.author_handle, t.data,
			coalesce((SELECT group_concat(json_extract(l.page, '$.title'), ' ')
				FROM json_each(t.data, '$.links') je JOIN links l ON l.url = json_extract(je.value, '$.expanded')), ''),
			coalesce((SELECT group_concat(ci.collection_id) FROM collection_items ci JOIN collections c ON c.id = ci.collection_id
				WHERE ci.tweet_id = t.id AND c.query = ''), '')
		FROM bookmarks b JOIN tweets t ON t.id = b.tweet_id
		WHERE b.removed_at IS NULL OR t.id = ?`, tweetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type doc struct {
		author string
		terms  map[string]float64
		in     []int64
	}
	var docs []doc
	var target *doc
	df := map[string]int{}
	for rows.Next() {
		var id, author, data, titles, cols string
		if err := rows.Scan(&id, &author, &data, &titles, &cols); err != nil {
			return nil, err
		}
		var t xparse.Tweet
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			return nil, err
		}
		text := t.Text + " " + titles
		if t.Quoted != nil {
			text += " " + t.Quoted.Text
		}
		d := doc{author: strings.ToLower(author), terms: termCounts(text)}
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

	// Each collection's centroid is the sum of its posts' unit vectors.
	centroid := map[int64]map[string]float64{}
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
