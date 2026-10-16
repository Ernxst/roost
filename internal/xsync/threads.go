package xsync

import (
	"context"
	"errors"
	"time"

	"roost/internal/store"
	"roost/internal/xparse"
)

// DetailPage is the X page thread capture drives: it opens one post at a
// time and collects X's TweetDetail responses for it.
type DetailPage interface {
	URL() (string, error)
	Load(url string) error
	// DrainDetail returns the bodies of TweetDetail responses received
	// since the last call.
	DrainDetail() ([]string, error)
	Scroll() error
}

// ErrSignedOut means X wants the user to sign in. Thread capture runs in
// the background, so it stops instead of asking.
var ErrSignedOut = errors.New("signed out of X")

// Thread opens a post and returns its thread, scrolling until X stops
// sending more of it. It returns nil when the post stands alone.
func Thread(ctx context.Context, page DetailPage, c store.ThreadCandidate, tick time.Duration) ([]xparse.Tweet, error) {
	if tick == 0 {
		tick = time.Second
	}
	page.DrainDetail()
	if err := page.Load(c.URL); err != nil {
		return nil, err
	}
	var bodies []string
	var chain []xparse.Tweet
	idle := 0
	for i := 0; i < 20 && idle < 3; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(tick):
		}
		if u, err := page.URL(); err == nil && isLogin(u) {
			return nil, ErrSignedOut
		}
		got, err := page.DrainDetail()
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, got...)
		if len(bodies) == 0 {
			continue
		}
		next, err := xparse.Thread(bodies, c.ID)
		if err != nil {
			return nil, err
		}
		if len(next) > len(chain) {
			chain, idle = next, 0
		} else {
			idle++
		}
		page.Scroll()
	}
	if len(bodies) == 0 {
		return nil, errors.New("X sent nothing for " + c.URL)
	}
	return chain, nil
}
