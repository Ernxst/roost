package xparse

import (
	"encoding/json"
	"testing"
)

func detailTweet(id, handle, replyTo string, minute int) map[string]any {
	return map[string]any{"tweet_results": map[string]any{"result": map[string]any{
		"rest_id": id,
		"core":    map[string]any{"user_results": map[string]any{"result": map[string]any{"core": map[string]any{"screen_name": handle, "name": handle}}}},
		"legacy": map[string]any{
			"full_text":                 "post " + id,
			"created_at":                "Wed Oct 15 10:0" + string(rune('0'+minute)) + ":00 +0000 2025",
			"in_reply_to_status_id_str": replyTo,
		},
	}}}
}

func TestThread(t *testing.T) {
	// ada posts 1, 2 and 3 as a thread; bob replies to 2; ada answers bob.
	body, _ := json.Marshal(map[string]any{"entries": []any{
		map[string]any{"entryId": "tweet-1", "content": map[string]any{"itemContent": detailTweet("1", "ada", "", 0)}},
		map[string]any{"entryId": "tweet-2", "content": map[string]any{"itemContent": detailTweet("2", "ada", "1", 1)}},
		map[string]any{"entryId": "conversationthread-3", "content": map[string]any{"items": []any{
			map[string]any{"item": map[string]any{"itemContent": detailTweet("3", "ada", "2", 2)}},
			map[string]any{"item": map[string]any{"itemContent": detailTweet("4", "bob", "2", 3)}},
			map[string]any{"item": map[string]any{"itemContent": detailTweet("5", "ada", "4", 4)}},
		}}},
	}})
	chain, err := Thread([]string{string(body)}, "2")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range chain {
		got = append(got, c.ID)
	}
	if len(got) != 3 || got[0] != "1" || got[1] != "2" || got[2] != "3" {
		t.Fatalf("thread = %v, want [1 2 3]", got)
	}
	if chain, _ := Thread([]string{string(body)}, "4"); chain != nil {
		t.Fatalf("bob's lone reply came back as a thread: %d posts", len(chain))
	}
}
