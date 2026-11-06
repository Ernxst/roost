package xparse

import (
	"encoding/json"
	"fmt"
)

// Thread reads the bodies of X's TweetDetail responses for the post focal
// and returns the thread it belongs to: the run of posts by the same author,
// each replying to the one before, in reading order. It returns nil when the
// post stands alone.
func Thread(bodies []string, focal string) ([]Tweet, error) {
	byID, err := detailTweets(bodies)
	if err != nil {
		return nil, err
	}
	start, ok := byID[focal]
	if !ok {
		return nil, nil
	}
	author := start.AuthorHandle
	chain := []Tweet{start}
	// Walk up to the first post, then down through the author's replies.
	for cur := start; ; {
		parent, ok := byID[cur.ReplyTo]
		if !ok || parent.AuthorHandle != author {
			break
		}
		chain = append([]Tweet{parent}, chain...)
		cur = parent
	}
	repliesTo := map[string]Tweet{}
	for _, t := range byID {
		if t.AuthorHandle == author && t.ReplyTo != "" {
			if prev, seen := repliesTo[t.ReplyTo]; !seen || t.CreatedAt.Before(prev.CreatedAt) {
				repliesTo[t.ReplyTo] = t
			}
		}
	}
	for cur := start; ; {
		next, ok := repliesTo[cur.ID]
		if !ok {
			break
		}
		chain = append(chain, next)
		cur = next
	}
	if len(chain) < 2 {
		return nil, nil
	}
	return chain, nil
}

// Post returns the post id as X's TweetDetail responses show it. Shown on
// its own, a post carries fields the bookmarks timeline leaves out, such as
// the whole text of its article.
func Post(bodies []string, id string) (Tweet, bool, error) {
	byID, err := detailTweets(bodies)
	if err != nil {
		return Tweet{}, false, err
	}
	t, ok := byID[id]
	return t, ok, nil
}

// detailTweets reads every post in TweetDetail responses, keyed by ID.
func detailTweets(bodies []string) (map[string]Tweet, error) {
	byID := map[string]Tweet{}
	for _, body := range bodies {
		var root any
		if err := json.Unmarshal([]byte(body), &root); err != nil {
			return nil, fmt.Errorf("decode post detail response: %w", err)
		}
		var walk func(any)
		walk = func(node any) {
			switch n := node.(type) {
			case []any:
				for _, v := range n {
					walk(v)
				}
			case map[string]any:
				if res, ok := n["tweet_results"].(map[string]any); ok {
					if t, ok := parseTweet(res["result"]); ok {
						t.Raw = nil
						// A later response may repeat the post without its article.
						if prev, seen := byID[t.ID]; !seen || !prev.hasArticleText() || t.hasArticleText() {
							byID[t.ID] = t
						}
					}
					return
				}
				for _, v := range n {
					walk(v)
				}
			}
		}
		walk(root)
	}
	return byID, nil
}
