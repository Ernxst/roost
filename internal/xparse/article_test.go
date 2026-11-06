package xparse

import "testing"

// articlePost is a post publishing an article, shaped as X's TweetDetail
// sends it. The article's editor blocks hold paragraphs, headings, list
// items and an image in an atomic block.
const articlePost = `{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"entries":[
  {"entryId":"tweet-9","content":{"itemContent":{"tweet_results":{"result":{
    "rest_id":"9",
    "core":{"user_results":{"result":{"core":{"screen_name":"ada","name":"Ada"}}}},
    "legacy":{"created_at":"Thu Oct 15 06:45:33 +0000 2026","full_text":"https://t.co/a",
      "entities":{"urls":[{"url":"https://t.co/a","expanded_url":"http://x.com/i/article/77","display_url":"x.com/i/article/77"}]}},
    "article":{"article_results":{"result":{
      "rest_id":"77","title":"Invert, always invert","preview_text":"A roadmap full of solutions",
      "content_state":{"entityMap":[{"key":"0","value":{"type":"MEDIA","data":{}}}],"blocks":[
        {"type":"unstyled","text":"A roadmap full of solutions."},
        {"type":"atomic","text":" ","entityRanges":[{"key":0,"offset":0,"length":1}]},
        {"type":"header-two","text":"The skill"},
        {"type":"unordered-list-item","text":"Name the assumption"},
        {"type":"unordered-list-item","text":"Say how to test it"},
        {"type":"unstyled","text":""},
        {"type":"ordered-list-item","text":"Read the tickets"},
        {"type":"ordered-list-item","text":"Ask five users"}
      ]}}}}
  }}}}}
]}]}}}`

func TestPostArticle(t *testing.T) {
	post, ok, err := Post([]string{articlePost}, "9")
	if err != nil || !ok {
		t.Fatalf("Post = %v, %v", ok, err)
	}
	if post.Text != "" {
		t.Errorf("text = %q, want the link to the article left out", post.Text)
	}
	a := post.Article
	if a == nil {
		t.Fatal("post has no article")
	}
	if a.URL != "http://x.com/i/article/77" || a.Title != "Invert, always invert" || a.Preview != "A roadmap full of solutions" {
		t.Errorf("article = %+v, want the linked URL, title and preview", a)
	}
	want := "A roadmap full of solutions.\n\nThe skill\n\n• Name the assumption\n\n• Say how to test it\n\n1. Read the tickets\n\n2. Ask five users"
	if a.Text != want {
		t.Errorf("text = %q\nwant %q", a.Text, want)
	}
}

func TestArticleWithoutLink(t *testing.T) {
	post, _ := ParseTweet([]byte(`{"rest_id":"9","legacy":{"full_text":""},
		"article":{"article_results":{"result":{"rest_id":"77","title":"Untitled"}}}}`))
	if post.Article == nil || post.Article.URL != "https://x.com/i/article/77" || post.Article.Text != "" {
		t.Fatalf("article = %+v, want X's address for it and no text", post.Article)
	}
	if len(post.Links) != 1 || post.Links[0].Expanded != "https://x.com/i/article/77" {
		t.Errorf("links = %+v, want a link to the article", post.Links)
	}
}
