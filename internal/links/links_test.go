package links

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetch(t *testing.T) {
	para := strings.Repeat("Shaders run once per pixel, in parallel, on the GPU. ", 12)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/article":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`<html><head><title>How shaders work</title></head><body>
				<nav>Home · About</nav><article><h1>How shaders work</h1><p>` + para + `</p><p>` + para + `</p></article>
				<script>alert(1)</script></body></html>`))
		case "/file.pdf":
			w.Header().Set("Content-Type", "application/pdf")
		}
	}))
	defer srv.Close()

	p := Fetch(context.Background(), srv.URL+"/article")
	if p.Error != "" || p.Title != "How shaders work" || !strings.Contains(p.Text, "once per pixel") {
		t.Fatalf("article = %+v", p)
	}
	if strings.Contains(p.Text, "alert") || strings.Contains(p.Text, "<") {
		t.Errorf("markup or script survived: %q", p.Text)
	}
	if p := Fetch(context.Background(), srv.URL+"/file.pdf"); !strings.Contains(p.Error, "not a web page") {
		t.Errorf("pdf = %+v", p)
	}
}

func TestSkip(t *testing.T) {
	for u, want := range map[string]bool{
		"https://x.com/a/status/1":   true,
		"https://www.twitter.com/a":  true,
		"mailto:a@b.c":               true,
		"https://github.com/a/b":     false,
		"http://blog.example.com/p/": false,
	} {
		if Skip(u) != want {
			t.Errorf("Skip(%q) = %v", u, !want)
		}
	}
}
