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

// schedule starts an incremental sync whenever the last one is older than
// the user's interval, checking once a minute.
func (l *Library) schedule() {
	for range time.Tick(time.Minute) {
		l.trayChanged()
		s, err := l.store()
		if err != nil {
			continue
		}
		set, err := s.Settings()
		if err != nil || set.SyncEvery <= 0 {
			continue
		}
		st, err := s.Stats()
		if err != nil {
			continue
		}
		if st.LastSync != nil && time.Since(st.LastSync.FinishedAt) < time.Duration(set.SyncEvery)*time.Minute {
			continue
		}
		if err := l.startSync(false, true); err == nil {
			log.Printf("sync: background sync started")
		}
	}
}

// notifyAdded tells the user about the bookmarks a background sync found.
func (l *Library) notifyAdded(s *store.Store, rec store.Sync) {
	if rec.Added == 0 {
		return
	}
	set, err := s.Settings()
	if err != nil || !set.Notify || !mygo.NotificationsSupported() {
		return
	}
	authors, _ := s.NewAuthors(rec.StartedAt)
	title := "1 new bookmark"
	if rec.Added > 1 {
		title = fmt.Sprintf("%d new bookmarks", rec.Added)
	}
	n := mygo.NewNotification(mygo.NotificationOptions{
		ID:    "sync",
		Title: title,
		Body:  byline(authors),
		Group: "sync",
	})
	if err := n.Show(); err != nil {
		log.Printf("notify: %v", err)
	}
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
