package xparse

import (
	"testing"
	"time"
)

const page = `{"data":{"bookmark_timeline_v2":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[
  {"entryId":"tweet-1","sortIndex":"100","content":{"itemContent":{"tweet_results":{"result":{
    "__typename":"Tweet","rest_id":"1",
    "core":{"user_results":{"result":{"core":{"screen_name":"ada","name":"Ada"},"avatar":{"image_url":"https://pbs.twimg.com/a.jpg"}}}},
    "legacy":{"created_at":"Thu Oct 15 06:45:33 +0000 2026","full_text":"look https://t.co/x https://t.co/m",
      "entities":{"urls":[{"url":"https://t.co/x","expanded_url":"https://example.com/x","display_url":"example.com/x"}]},
      "extended_entities":{"media":[{"type":"video","url":"https://t.co/m","media_url_https":"https://pbs.twimg.com/v.jpg",
        "original_info":{"width":1280,"height":720},
        "video_info":{"variants":[{"content_type":"video/mp4","bitrate":256,"url":"low.mp4"},{"content_type":"video/mp4","bitrate":2176,"url":"high.mp4"},{"content_type":"application/x-mpegURL","url":"x.m3u8"}]}}]}},
    "quoted_status_result":{"result":{"rest_id":"2","core":{"user_results":{"result":{"legacy":{"screen_name":"bob","name":"Bob"}}}},
      "legacy":{"created_at":"Mon Jun 28 04:13:47 +0000 2021","full_text":"quoted"}}}
  }}}}},
  {"entryId":"tweet-3","sortIndex":"200","content":{"itemContent":{"tweet_results":{"result":{
    "__typename":"TweetWithVisibilityResults","tweet":{"rest_id":"3",
      "core":{"user_results":{"result":{"core":{"screen_name":"cy","name":"Cy"}}}},
      "note_tweet":{"note_tweet_results":{"result":{"text":"a long post","entity_set":{"urls":[]}}}},
      "legacy":{"created_at":"Mon Jun 28 04:13:47 +0000 2021","full_text":"a long…"}}}}}}},
  {"entryId":"cursor-bottom-1","content":{"value":"abc"}}
]}]}}}}`

func TestParse(t *testing.T) {
	entries, err := Parse([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Tweet.ID != "3" || entries[1].Tweet.ID != "1" {
		t.Fatalf("entries not ordered newest bookmark first: %s, %s", entries[0].Tweet.ID, entries[1].Tweet.ID)
	}

	long := entries[0].Tweet
	if long.Text != "a long post" || long.AuthorHandle != "cy" {
		t.Errorf("long post = %q by %q, want the note text by cy", long.Text, long.AuthorHandle)
	}

	tw := entries[1].Tweet
	if tw.URL != "https://x.com/ada/status/1" || tw.AuthorName != "Ada" || tw.AuthorAvatar != "https://pbs.twimg.com/a.jpg" {
		t.Errorf("author fields wrong: %+v", tw)
	}
	if tw.Text != "look https://t.co/x" {
		t.Errorf("text = %q, want the media link stripped", tw.Text)
	}
	if !tw.CreatedAt.Equal(time.Date(2026, 10, 15, 6, 45, 33, 0, time.UTC)) {
		t.Errorf("createdAt = %v", tw.CreatedAt)
	}
	if len(tw.Links) != 1 || tw.Links[0].Expanded != "https://example.com/x" {
		t.Errorf("links = %+v", tw.Links)
	}
	if len(tw.Media) != 1 || tw.Media[0].Video != "high.mp4" || tw.Media[0].Width != 1280 {
		t.Errorf("media = %+v, want the highest-bitrate mp4", tw.Media)
	}
	if tw.Quoted == nil || tw.Quoted.AuthorHandle != "bob" || tw.Quoted.Text != "quoted" {
		t.Errorf("quoted = %+v", tw.Quoted)
	}
}

func TestParseEndOfTimeline(t *testing.T) {
	entries, err := Parse([]byte(`{"data":{"bookmark_timeline_v2":{"timeline":{"instructions":[{"entries":[{"entryId":"cursor-top-1"},{"entryId":"cursor-bottom-1"}]}]}}}}`))
	if err != nil || len(entries) != 0 {
		t.Fatalf("got %d entries, err %v; want none", len(entries), err)
	}
}

func TestParseTranslation(t *testing.T) {
	const post = `{"rest_id":"4","core":{"user_results":{"result":{"core":{"screen_name":"jp","name":"JP"}}}},
	  "legacy":{"created_at":"Mon Jun 28 04:13:47 +0000 2021","full_text":"見て https://t.co/m","lang":"ja",
	    "extended_entities":{"media":[{"type":"photo","url":"https://t.co/m","media_url_https":"https://pbs.twimg.com/p.jpg"}]}},
	  "grok_translated_post_with_availability":{"is_available":true,"data":{"source_language":"ja","destination_language":"en","translation":"Look https://t.co/m"}}}`
	tw, ok := ParseTweet([]byte(post))
	if !ok {
		t.Fatal("post not parsed")
	}
	want := Translation{Text: "Look", From: "ja", To: "en"}
	if tw.Translation == nil || *tw.Translation != want {
		t.Errorf("translation = %+v, want %+v with the media link stripped", tw.Translation, want)
	}

	const english = `{"rest_id":"5","core":{"user_results":{"result":{"core":{"screen_name":"en"}}}},
	  "legacy":{"created_at":"Mon Jun 28 04:13:47 +0000 2021","full_text":"hello","lang":"en"},
	  "grok_translated_post_with_availability":{"is_available":false}}`
	if tw, _ := ParseTweet([]byte(english)); tw.Translation != nil {
		t.Errorf("a post in the reader's language got translation %+v", tw.Translation)
	}
}
