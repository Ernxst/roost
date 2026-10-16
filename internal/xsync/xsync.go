// Package xsync brings the library up to date with the user's bookmarks on X.
//
// X's own web app does the fetching: a page loads x.com/i/bookmarks with a
// script that copies every Bookmarks GraphQL response into a queue, and the
// sync scrolls that page and drains the queue. Requests therefore carry
// whatever headers and tokens X currently requires, and the app never handles
// the user's credentials.
package xsync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"roost/internal/store"
	"roost/internal/xparse"
)

// BookmarksURL is where the sync starts. X redirects it to its History page,
// whose first tab lists bookmarks.
const BookmarksURL = "https://x.com/i/bookmarks"

// Page is the X page a sync drives.
type Page interface {
	URL() (string, error)
	// Drain returns the bodies of Bookmarks responses received since the
	// last call.
	Drain() ([]string, error)
	Scroll() error
	// ShowLogin shows the page so the user can sign in to X; HideLogin
	// hides it again once they have.
	ShowLogin()
	HideLogin()
}

// Phase is the stage a sync has reached.
type Phase string

const (
	PhaseLoading Phase = "loading"
	PhaseLogin   Phase = "login"
	PhaseReading Phase = "reading"
	PhaseDone    Phase = "done"
	PhaseFailed  Phase = "failed"
)

type Progress struct {
	Phase   Phase  `json:"phase"`
	Full    bool   `json:"full"`
	Seen    int    `json:"seen"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Message string `json:"message"`
}

type Options struct {
	// Full reads the whole timeline and marks bookmarks missing from it as
	// removed. Otherwise the sync stops at the first bookmark the library
	// already holds.
	Full bool
	// Tick is the pause between scrolls; Stall is how long the sync waits
	// without a new response before it stops; LoginTimeout is how long it
	// waits for the user to sign in.
	Tick, Stall, LoginTimeout time.Duration
	Now                       func() time.Time
	Progress                  func(Progress)
}

func (o *Options) defaults() {
	if o.Tick == 0 {
		o.Tick = 1200 * time.Millisecond
	}
	if o.Stall == 0 {
		o.Stall = 30 * time.Second
	}
	if o.LoginTimeout == 0 {
		o.LoginTimeout = 10 * time.Minute
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Progress == nil {
		o.Progress = func(Progress) {}
	}
}

// Run syncs the library from page and records the outcome in the store.
func Run(ctx context.Context, page Page, lib *store.Store, opts Options) (store.Sync, error) {
	opts.defaults()
	rec := store.Sync{Full: opts.Full, StartedAt: opts.Now()}
	err := run(ctx, page, lib, opts, &rec)
	rec.FinishedAt = opts.Now()
	p := Progress{Phase: PhaseDone, Full: rec.Full, Seen: rec.Seen, Added: rec.Added, Removed: rec.Removed}
	if err != nil {
		rec.Error = err.Error()
		p.Phase, p.Message = PhaseFailed, err.Error()
	} else if !rec.ReachedEnd && opts.Full {
		p.Message = "X stopped sending bookmarks before the end, so none were marked as removed."
	}
	if recErr := lib.RecordSync(rec); recErr != nil && err == nil {
		err = recErr
	}
	opts.Progress(p)
	return rec, err
}

func run(ctx context.Context, page Page, lib *store.Store, opts Options, rec *store.Sync) error {
	opts.Progress(Progress{Phase: PhaseLoading, Full: opts.Full})
	lastResponse := opts.Now()
	var loginSince time.Time
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.Tick):
		}

		url, err := page.URL()
		if err != nil {
			return fmt.Errorf("read the X page address: %w", err)
		}
		if isLogin(url) {
			if loginSince.IsZero() {
				loginSince = opts.Now()
				page.ShowLogin()
				opts.Progress(Progress{Phase: PhaseLogin, Full: opts.Full})
			}
			if opts.Now().Sub(loginSince) > opts.LoginTimeout {
				page.HideLogin()
				return errors.New("timed out waiting for you to sign in to X")
			}
			continue
		}
		if !loginSince.IsZero() {
			loginSince = time.Time{}
			lastResponse = opts.Now()
			page.HideLogin()
		}

		bodies, err := page.Drain()
		if err != nil {
			return fmt.Errorf("read responses from the X page: %w", err)
		}
		stop := false
		for _, body := range bodies {
			entries, err := xparse.Parse([]byte(body))
			if err != nil {
				return err
			}
			lastResponse = opts.Now()
			if len(entries) == 0 {
				rec.ReachedEnd = true
				stop = true
				continue
			}
			added, err := lib.Ingest(entries, rec.StartedAt)
			if err != nil {
				return err
			}
			rec.Seen += len(entries)
			rec.Added += added
			if !opts.Full && added < len(entries) {
				stop = true
			}
		}
		if len(bodies) > 0 {
			opts.Progress(Progress{Phase: PhaseReading, Full: opts.Full, Seen: rec.Seen, Added: rec.Added})
		}
		if stop {
			break
		}
		if opts.Now().Sub(lastResponse) > opts.Stall {
			if rec.Seen == 0 {
				return errors.New("X sent no bookmarks; check that the X page loads")
			}
			break
		}
		if err := page.Scroll(); err != nil {
			return fmt.Errorf("scroll the X page: %w", err)
		}
	}

	if opts.Full && rec.ReachedEnd {
		removed, err := lib.MarkRemoved(rec.StartedAt, opts.Now())
		if err != nil {
			return err
		}
		rec.Removed = removed
	}
	return nil
}

func isLogin(url string) bool {
	return strings.Contains(url, "/login") || strings.Contains(url, "/i/flow/") || strings.Contains(url, "/onboarding")
}
