package media

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestAllowed(t *testing.T) {
	for u, want := range map[string]bool{
		"https://pbs.twimg.com/media/a.jpg?name=small": true,
		"https://video.twimg.com/ext_tw_video/1/a.mp4": true,
		"http://pbs.twimg.com/media/a.jpg":             false,
		"https://example.com/a.jpg":                    false,
		"https://pbs.twimg.com.evil.com/a.jpg":         false,
	} {
		if Allowed(u) != want {
			t.Errorf("Allowed(%q) = %v", u, !want)
		}
	}
}

func TestFetchStoresOnce(t *testing.T) {
	var hits atomic.Int32
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		io.WriteString(w, "jpegbytes")
	}))
	defer origin.Close()

	// Send requests for X's hosts to the test server.
	tr := origin.Client().Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return new(net.Dialer).DialContext(ctx, network, origin.Listener.Addr().String())
	}
	tr.TLSClientConfig.InsecureSkipVerify = true
	c := New(t.TempDir())
	c.client = &http.Client{Transport: tr}

	u := "https://pbs.twimg.com/media/a.jpg?name=small"
	if n := c.Prefetch(context.Background(), []string{u, u, "https://example.com/x.jpg"}); n < 1 {
		t.Fatalf("prefetched %d", n)
	}
	if !c.Has(u) {
		t.Fatal("not cached")
	}

	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, httptest.NewRequest("GET", "roost-media://localhost/?u="+url.QueryEscape(u), nil))
	if rec.Body.String() != "jpegbytes" || hits.Load() != 1 {
		t.Fatalf("served %q after %d origin hits", rec.Body.String(), hits.Load())
	}
	rec = httptest.NewRecorder()
	c.ServeHTTP(rec, httptest.NewRequest("GET", "roost-media://localhost/?u="+url.QueryEscape("https://example.com/x.jpg"), nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("other host got %d", rec.Code)
	}
	if files, bytes, _ := c.Usage(); files != 1 || bytes != 9 {
		t.Fatalf("usage %d files, %d bytes", files, bytes)
	}
}
