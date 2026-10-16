package main

import "github.com/egoist/mygo"

// capture runs in x.com before X's own scripts and copies the body of every
// Bookmarks GraphQL response into window.__roost, and every TweetDetail
// response into window.__roostDetail, for Go to drain.
const capture = `(() => {
  if (!/(^|\.)x\.com$/.test(location.hostname)) return;
  const queues = { Bookmarks: (window.__roost = []), TweetDetail: (window.__roostDetail = []) };
  const queueFor = (url) => queues[(/\/graphql\/[^/]+\/(\w+)(\?|$)/.exec(String(url)) || [])[1]];
  const fetch = window.fetch;
  window.fetch = async function (input, init) {
    const res = await fetch.apply(this, arguments);
    const url = typeof input === "string" ? input : input && input.url;
    const queue = queueFor(url);
    if (queue) res.clone().text().then((t) => queue.push(t), () => {});
    return res;
  };
  const open = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function (method, url) {
    this.__roostUrl = url;
    return open.apply(this, arguments);
  };
  const send = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.send = function () {
    this.addEventListener("load", () => {
      const queue = queueFor(this.__roostUrl);
      if (queue && typeof this.responseText === "string") queue.push(this.responseText);
    });
    return send.apply(this, arguments);
  };
})();`

// syncPage is the window a sync drives. It stays hidden unless the user has
// to sign in to X.
type syncPage struct{ win *mygo.Window }

func openSyncPage(parent *mygo.Window, url string) *syncPage {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:  "Sign in to X for roost",
		URL:    url,
		Width:  520,
		Height: 760,
		Hidden: true,
		Parent: parent,
		Page:   mygo.PageOptions{PreloadScript: capture},
	})
	return &syncPage{win: win}
}

func (p *syncPage) URL() (string, error) { return p.win.Page().URL(), nil }

func (p *syncPage) Drain() ([]string, error) {
	return mygo.EvalAs[[]string](p.win.Page(), "return (window.__roost || []).splice(0)")
}

func (p *syncPage) DrainDetail() ([]string, error) {
	return mygo.EvalAs[[]string](p.win.Page(), "return (window.__roostDetail || []).splice(0)")
}

func (p *syncPage) Load(url string) error {
	return p.win.Page().LoadURL(url)
}

func (p *syncPage) Scroll() error {
	_, err := p.win.Page().Eval("window.scrollBy(0, innerHeight * 3)")
	return err
}

func (p *syncPage) ShowLogin() { p.win.Show(); p.win.Focus() }
func (p *syncPage) HideLogin() { p.win.Hide() }
func (p *syncPage) Close()     { p.win.Destroy() }

// quietPage is a sync page that never shows itself to ask for a sign-in.
type quietPage struct{ *syncPage }

func (quietPage) ShowLogin() {}
