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
						byID[t.ID] = t
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
