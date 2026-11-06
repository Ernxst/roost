package xparse

import "testing"

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
