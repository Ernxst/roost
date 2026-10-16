package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"

	"roost/internal/store"
)

// menuBar is roost's menu bar icon, which keeps the app reachable while its
// window is closed and background syncs run.
type menuBar struct {
	tray   *mygo.Tray
	sync   *mygo.MenuItem
	status *mygo.MenuItem
}

// stayInMenuBar keeps roost running when its window closes: closing hides
// the window, and the menu bar icon or the Dock brings it back.
func (l *Library) stayInMenuBar() {
	var quitting atomic.Bool
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting.Store(true) })
	mygo.App.OnWindowAllClosed(func() {})
	l.main.OnClose(func(e *mygo.CloseEvent) {
		if !quitting.Load() {
			e.PreventDefault()
			l.main.Hide()
		}
	})
	mygo.App.OnActivate(func(visible bool) {
		if !visible {
			l.show()
		}
	})

	m := &menuBar{
		sync:   &mygo.MenuItem{Label: "Sync now", Click: func(*mygo.MenuItem, *mygo.Window) { l.syncNow() }},
		status: &mygo.MenuItem{Label: "Not synced yet", Disabled: true},
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open roost", Click: func(*mygo.MenuItem, *mygo.Window) { l.show() }},
		mygo.Separator(),
		m.sync,
		m.status,
		mygo.Separator(),
		updater.MenuItem(),
		{Label: "Quit roost", Role: mygo.RoleQuit},
	})
	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: "roost", Menu: menu})
	if err != nil {
		log.Printf("menu bar: %v", err)
		return
	}
	m.tray = tray
	l.mu.Lock()
	l.tray = m
	l.mu.Unlock()
	l.trayChanged()
}

func (l *Library) show() {
	l.main.Show()
	l.main.Focus()
}

func (l *Library) syncNow() {
	if err := l.startSync(false, false); err != nil {
		log.Printf("sync: %v", err)
	}
}

// trayChanged brings the menu bar's sync item and status up to date.
func (l *Library) trayChanged() {
	l.mu.Lock()
	m, syncing, s := l.tray, l.syncing, l.lib
	l.mu.Unlock()
	if m == nil || s == nil {
		return
	}
	m.sync.SetEnabled(!syncing)
	if syncing {
		m.status.SetLabel("Syncing…")
		return
	}
	st, err := s.Stats()
	if err != nil || st.LastSync == nil {
		return
	}
	m.status.SetLabel(syncStatus(*st.LastSync, time.Now()))
}

func syncStatus(r store.Sync, now time.Time) string {
	if r.Error != "" {
		return "Last sync failed " + ago(r.FinishedAt, now)
	}
	return "Synced " + ago(r.FinishedAt, now)
}

func ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return t.Format("2 Jan")
}

// fullSyncEvery is how often a background sync reads the whole timeline, the
// only way to notice bookmarks that have gone from X.
const fullSyncEvery = 7 * 24 * time.Hour

// schedule starts a background sync whenever the last one is older than the
// user's interval, checking once a minute.
func (l *Library) schedule() {
	for range time.Tick(time.Minute) {
		l.trayChanged()
		s, err := l.store()
		if err != nil {
			continue
		}
		set, err := s.Settings()
		if err != nil {
			continue
		}
		st, err := s.Stats()
		if err != nil {
			continue
		}
		lastFull, err := s.LastFullSync()
		if err != nil {
			continue
		}
		due, full := dueSync(st.LastSync, lastFull, set.SyncEvery, time.Now())
		if !due {
			continue
		}
		if err := l.startSync(full, true); err == nil {
			log.Printf("sync: background sync started (full: %v)", full)
		}
	}
}

// dueSync decides whether a background sync is due, every minutes after the
// last one, and whether it should read the whole timeline because the last
// complete read finished more than fullSyncEvery ago.
func dueSync(last *store.Sync, lastFull time.Time, every int, now time.Time) (due, full bool) {
	if every <= 0 {
		return false, false
	}
	if last != nil && now.Sub(last.FinishedAt) < time.Duration(every)*time.Minute {
		return false, false
	}
	return true, !lastFull.IsZero() && now.Sub(lastFull) >= fullSyncEvery
}

// notifySync tells the user what a background sync found.
func (l *Library) notifySync(s *store.Store, rec store.Sync) {
	set, err := s.Settings()
	if err != nil || !set.Notify || !mygo.NotificationsSupported() {
		return
	}
	var authors []string
	if rec.Added > 0 {
		authors, _ = s.NewAuthors(rec.StartedAt)
	}
	title, body := syncNotice(rec, authors)
	if title == "" {
		return
	}
	n := mygo.NewNotification(mygo.NotificationOptions{ID: "sync", Title: title, Body: body, Group: "sync"})
	if err := n.Show(); err != nil {
		log.Printf("notify: %v", err)
	}
}

// syncNotice words the notification for a sync that added and removed
// bookmarks, given the new bookmarks' authors; an empty title means there is
// nothing to tell.
func syncNotice(rec store.Sync, authors []string) (title, body string) {
	var gone string
	switch {
	case rec.Removed == 1:
		gone = "1 bookmark is gone from X; roost kept its copy."
	case rec.Removed > 1:
		gone = fmt.Sprintf("%d bookmarks are gone from X; roost kept their copies.", rec.Removed)
	}
	switch {
	case rec.Added == 1:
		title = "1 new bookmark"
	case rec.Added > 1:
		title = fmt.Sprintf("%d new bookmarks", rec.Added)
	case rec.Removed == 1:
		return "1 bookmark gone from X", "roost kept its copy."
	case rec.Removed > 1:
		return fmt.Sprintf("%d bookmarks gone from X", rec.Removed), "roost kept their copies."
	default:
		return "", ""
	}
	return title, strings.TrimSpace(byline(authors) + "\n" + gone)
}

// byline names up to three authors, such as "From @ada, @grace and 2 others".
func byline(handles []string) string {
	if len(handles) == 0 {
		return ""
	}
	shown := handles[:min(3, len(handles))]
	names := make([]string, len(shown))
	for i, h := range shown {
		names[i] = "@" + h
	}
	switch rest := len(handles) - len(shown); {
	case rest == 1:
		return "From " + strings.Join(names, ", ") + " and 1 other"
	case rest > 1:
		return fmt.Sprintf("From %s and %d others", strings.Join(names, ", "), rest)
	case len(names) == 1:
		return "From " + names[0]
	}
	return "From " + strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// trayIcon draws a bookmark ribbon as a 32×32 template image.
func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{A: 255}
	for y := 4; y < 28; y++ {
		for x := 9; x < 23; x++ {
			// The notch: a V cut up from the bottom edge to y = 20.
			notch := y >= 20 && abs(x-16)*8 < (y-20)*7+1
			if !notch {
				img.Set(x, y, ink)
			}
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
