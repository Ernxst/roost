import { useComputed, useSignal, useSignalEffect } from "@preact/signals";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";
import { Library, type Bookmark, type Collection } from "./mygo";
import { fixtureHref, fixtureMode, isFixtureMode, type FixtureMode } from "./fixture-mode";
import {
  announce,
  announcement,
  authors,
  collections,
  createCollection,
  cursor,
  dismissNotice,
  errorText,
  fail,
  find,
  items,
  loading,
  loadMore,
  manual,
  markOpened,
  mediaOnly,
  notice,
  opened,
  progress,
  reload,
  refreshMeta,
  saveNote,
  search,
  startSync,
  stats,
  toggleCollection,
  total,
  view,
  type View,
} from "./state";
import { Card, ink } from "./card";
import { RevisitView } from "./revisit";
import { SettingsButton } from "./settings";
import { SortMode } from "./sort";
import { SuggestedChips, useSuggestions } from "./suggest";
import { Thread } from "./thread";
import { Links, Reader } from "./links";
import { HighlightList, Highlighter, HighlightsView, usePostHighlights } from "./highlights";
import { DuplicatesView, WallView } from "./wall";
import { WhatsNew } from "./changelog";
import { Avatar, MediaGrid, openExternal, previewRatio, Quoted, relative, Text } from "./tweet";

const same = (a: View, b: View) => JSON.stringify(a) === JSON.stringify(b);

export function App() {
  useKeyboard();
  return (
    <div class="shell">
      <a class="skip" href="#feed">
        Skip to bookmarks
      </a>
      <Rail />
      <main class="desk" id="feed" tabIndex={-1}>
        <Toolbar />
        {view.value.kind === "sort" ? (
          <SortMode />
        ) : view.value.kind === "revisit" ? (
          <RevisitView />
        ) : view.value.kind === "highlights" ? (
          <HighlightsView />
        ) : view.value.kind === "wall" ? (
          <WallView />
        ) : view.value.kind === "duplicates" ? (
          <DuplicatesView />
        ) : (
          <Feed />
        )}
      </main>
      <Sheet />
      <div class="sr-only" role="status" aria-live="polite">
        {announcement.value}
      </div>
      <Notice />
      {import.meta.env.DEV && <FixtureSwitcher />}
    </div>
  );
}

/** The live region's message, shown for sighted readers too. */
function Notice() {
  const n = notice.value;
  return (
    <div class={n.error ? "notice error" : "notice"} data-shown={n.shown || undefined} aria-hidden={!n.shown || undefined}>
      <p>{n.text}</p>
      {n.error && (
        <button class="close" onClick={dismissNotice} aria-label="Dismiss" tabIndex={n.shown ? 0 : -1}>
          ✕
        </button>
      )}
    </div>
  );
}

const FIXTURE_MODES: { mode: FixtureMode; label: string }[] = [
  { mode: "demo", label: "Demo data" },
  { mode: "worst", label: "Worst case" },
  { mode: "empty", label: "Empty" },
  { mode: "one", label: "One" },
  { mode: "thousand", label: "1,000 rows" },
];

function FixtureSwitcher() {
  return (
    <nav class="fixture-switcher" aria-label="Development data fixtures">
      {FIXTURE_MODES.map(({ mode, label }) => (
        <a key={mode} href={fixtureHref(mode)} aria-current={fixtureMode === mode ? "page" : undefined}>
          {label}
        </a>
      ))}
    </nav>
  );
}

function RailItem({
  target,
  label,
  count,
  hue,
  smart,
  disabled = false,
}: {
  target: View;
  label: string;
  count?: number;
  hue?: number;
  smart?: string;
  disabled?: boolean;
}) {
  const current = same(view.value, target);
  return (
    <li>
      <button
        class="rail-item"
        aria-current={current ? "page" : undefined}
        disabled={disabled}
        onClick={() => (view.value = target)}
        style={hue === undefined ? undefined : { "--hue": ink(hue) }}
        title={hue === undefined ? undefined : smart ? `${label}\nSmart collection: ${smart}` : label}
      >
        {hue !== undefined && <span class={smart ? "tab smart" : "tab"} aria-hidden="true" />}
        <span class="label">{label}</span>
        {count !== undefined && <span class="count">{count.toLocaleString()}</span>}
      </button>
    </li>
  );
}

function Rail() {
  const s = stats.value;
  const adding = useSignal(false);
  const name = useSignal("");
  const error = useSignal("");

  async function submit(e: Event) {
    e.preventDefault();
    try {
      const c = await createCollection(name.value);
      name.value = "";
      error.value = "";
      adding.value = false;
      view.value = { kind: "collection", id: c.id };
    } catch (err) {
      error.value = errorText(err);
    }
  }

  return (
    <nav class="rail" aria-label="Library">
      <div class="masthead">
        <span class="wordmark">roost</span>
        <span class="tagline">a field journal of the posts you kept</span>
      </div>

      <ul class="rail-list">
        <RailItem target={{ kind: "all" }} label="Everything" count={s?.total} />
        <RailItem target={{ kind: "unsorted" }} label="Unsorted" count={s?.unsorted} disabled={isFixtureMode} />
        {!!s?.removed && <RailItem target={{ kind: "removed" }} label="Gone from X" count={s.removed} disabled={isFixtureMode} />}
      </ul>

      <h2 class="rail-heading">Rituals</h2>
      <ul class="rail-list">
        <RailItem target={{ kind: "sort" }} label="Sort the unsorted" count={s?.toSort} disabled={isFixtureMode} />
        <RailItem target={{ kind: "revisit" }} label="Revisit" disabled={isFixtureMode} />
        <RailItem target={{ kind: "highlights" }} label="Highlights" disabled={isFixtureMode} />
        <RailItem target={{ kind: "wall" }} label="Media wall" disabled={isFixtureMode} />
        <RailItem target={{ kind: "duplicates" }} label="Duplicates" disabled={isFixtureMode} />
      </ul>

      <h2 class="rail-heading">Collections</h2>
      <ul class="rail-list">
        {collections.value.map((c) => (
          <RailItem
            key={c.id}
            target={{ kind: "collection", id: c.id }}
            label={c.name}
            count={c.count}
            hue={c.hue}
            smart={c.query || undefined}
            disabled={isFixtureMode}
          />
        ))}
      </ul>
      {adding.value ? (
        <form class="new-collection" onSubmit={submit}>
          <label for="new-collection" class="field-label">
            New collection
          </label>
          <input
            id="new-collection"
            value={name.value}
            onInput={(e) => (name.value = e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Escape" && (adding.value = false)}
            placeholder="Name it…"
            autoComplete="off"
            aria-invalid={error.value ? "true" : undefined}
            aria-describedby={error.value ? "new-collection-error" : undefined}
            ref={(el) => el?.focus()}
          />
          {error.value && (
            <p id="new-collection-error" class="field-error">
              {error.value}
            </p>
          )}
        </form>
      ) : (
          <button class="add" disabled={isFixtureMode} onClick={() => (adding.value = true)}>
          + New collection
        </button>
      )}

      {authors.value.length > 0 && (
        <>
          <h2 class="rail-heading">Most kept</h2>
          <ul class="rail-list authors">
            {authors.value.map((a) => (
              <li key={a.handle}>
                <button
                  class="rail-item"
                  aria-current={same(view.value, { kind: "author", handle: a.handle }) ? "page" : undefined}
                  disabled={isFixtureMode}
                  onClick={() => (view.value = { kind: "author", handle: a.handle })}
                  title={a.name || a.handle}
                >
                  <Avatar src={a.avatar} size={18} />
                  <span class="label">{a.name || a.handle}</span>
                  <span class="count">{a.count.toLocaleString()}</span>
                </button>
              </li>
            ))}
          </ul>
        </>
      )}
      <div class="rail-foot">
        {!isFixtureMode && <SettingsButton />}
        {!isFixtureMode && <WhatsNew />}
      </div>
    </nav>
  );
}

function SyncStatus() {
  const p = progress.value;
  const last = stats.value?.lastSync;
  if (p && (p.phase === "loading" || p.phase === "reading")) {
    return (
      <span class="ticker live">
        <span class="pulse" aria-hidden="true" />
        {p.phase === "loading" ? "Opening X…" : `Reading · ${p.seen.toLocaleString()} seen · ${p.added.toLocaleString()} new`}
      </span>
    );
  }
  if (p?.phase === "login") return <span class="ticker live">Sign in to X in the window that opened</span>;
  if (p?.phase === "failed") return <span class="ticker failed">Sync failed: {p.message}</span>;
  if (!last) return <span class="ticker">Never synced</span>;
  return (
    <span class="ticker" title={new Date(last.finishedAt).toLocaleString()}>
      Synced {relative(last.finishedAt)} · {last.added.toLocaleString()} new
      {last.removed ? ` · ${last.removed.toLocaleString()} gone` : ""}
      {p?.phase === "done" && p.message ? ` · ${p.message}` : ""}
    </span>
  );
}

function Toolbar() {
  const busy = useComputed(() => {
    const ph = progress.value?.phase;
    return ph === "loading" || ph === "reading" || ph === "login";
  });
  const input = useRef<HTMLInputElement>(null);
  const focused = useSignal(false);
  return (
    <header class="toolbar">
      <div class="search" onFocusIn={() => (focused.value = true)} onFocusOut={() => (focused.value = false)}>
        <label for="search" class="sr-only">
          Search bookmarks
        </label>
        <span class="glyph" aria-hidden="true">
          ⌕
        </span>
        <input
          id="search"
          ref={input}
          type="search"
          value={search.value}
          onInput={(e) => (search.value = e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              search.value = "";
              input.current?.blur();
            }
          }}
          placeholder={`Search ${stats.value?.total.toLocaleString() ?? ""} kept posts`}
          autoComplete="off"
          spellcheck={false}
          aria-describedby="search-hints"
        />
        <kbd aria-hidden="true">/</kbd>
        <SearchHints open={focused.value && !search.value} input={input} />
      </div>
      {search.value.trim() && !isFixtureMode && <SaveSearch />}
      <button class="chip" disabled={isFixtureMode} aria-pressed={mediaOnly.value} onClick={() => (mediaOnly.value = !mediaOnly.value)}>
        With media
      </button>
      <div class="sync">
        <SyncStatus />
        <div class="sync-buttons">
          <button class="press" disabled={busy.value || isFixtureMode} onClick={() => startSync(false)}>
            Sync
          </button>
          <button
            class="press quiet"
            disabled={busy.value || isFixtureMode}
            onClick={() => startSync(true)}
            title="Read every bookmark and mark the ones gone from X"
          >
            Full
          </button>
        </div>
      </div>
    </header>
  );
}

const HINTS: [string, string][] = [
  ["from:", "posts by an author"],
  ["has:video", "also photo, gif, link, quote, note, thread"],
  ["in:", "in a collection; in:none for unsorted"],
  ["-", "leave out a word or filter"],
  ["before:2025-01-01", "or after:, by post date"],
  ["is:gone", "removed on X"],
];

function SearchHints({ open, input }: { open: boolean; input: { current: HTMLInputElement | null } }) {
  return (
    <div id="search-hints" class="hints" hidden={!open}>
      <p class="hints-title">Narrow it down</p>
      <ul>
        {HINTS.map(([op, what]) => (
          <li key={op}>
            <button
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                search.value = op;
                input.current?.focus();
              }}
            >
              <code>{op}</code>
              <span>{what}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function SaveSearch() {
  const naming = useSignal(false);
  const name = useSignal("");
  const error = useSignal("");
  async function save(e: Event) {
    e.preventDefault();
    try {
      const c = await createCollection(name.value, search.value);
      naming.value = false;
      name.value = "";
      error.value = "";
      search.value = "";
      view.value = { kind: "collection", id: c.id };
    } catch (err) {
      error.value = errorText(err);
    }
  }
  if (!naming.value) {
    return (
      <button class="chip" onClick={() => (naming.value = true)}>
        Save search
      </button>
    );
  }
  return (
    <form class="save-search" onSubmit={save}>
      <label for="save-search" class="field-label">
        Save search as
      </label>
      <input
        id="save-search"
        value={name.value}
        onInput={(e) => ((name.value = e.currentTarget.value), (error.value = ""))}
        onKeyDown={(e) => e.key === "Escape" && (naming.value = false)}
        placeholder="Name this search…"
        autoComplete="off"
        aria-invalid={error.value ? "true" : undefined}
        aria-describedby={error.value ? "save-search-error" : undefined}
        ref={(el) => el?.focus()}
      />
      {error.value && (
        <p id="save-search-error" class="field-error">
          {error.value}
        </p>
      )}
    </form>
  );
}

function viewTitle(v: View): string {
  switch (v.kind) {
    case "all":
      return "Everything";
    case "unsorted":
      return "Unsorted";
    case "removed":
      return "Gone from X";
    case "collection":
      return collections.value.find((c) => c.id === v.id)?.name ?? "Collection";
    case "sort":
      return "Sorting";
    case "revisit":
      return "Revisit";
    case "highlights":
      return "Highlights";
    case "wall":
      return "Media wall";
    case "duplicates":
      return "Duplicates";
    case "author": {
      const a = authors.value.find((a) => a.handle === v.handle);
      return a?.name || `@${v.handle}`;
    }
  }
}

// Columns are packed by estimated height so appending a page never moves
// cards that are already placed.
function estimate(b: Bookmark, width: number) {
  const t = b.tweet;
  const lines = (s: string) => Math.ceil(s.length / (width / 8.4)) + (s.match(/\n/g)?.length ?? 0);
  let h = 96 + lines(t.text) * 23;
  const m = t.media[0];
  const ratio = m && t.media.length === 1 ? previewRatio(m) : undefined;
  if (m) h += ratio ? Math.min(width / ratio, 520) : width * 0.6;
  if (t.quoted) h += 70 + lines(t.quoted.text) * 21 + (t.quoted.media.length ? width * 0.5 : 0);
  if (b.note) h += 12 + Math.min(lines(b.note), 4) * 20;
  if (b.collections.length) h += 30;
  return h;
}

function Feed() {
  const desk = useRef<HTMLDivElement>(null);
  const sentinel = useRef<HTMLDivElement>(null);
  const width = useSignal(900);

  useLayoutEffect(() => {
    const el = desk.current!;
    const ro = new ResizeObserver(([e]) => (width.value = e!.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  useEffect(() => {
    const io = new IntersectionObserver((es) => es[0]?.isIntersecting && loadMore(), { rootMargin: "1200px" });
    io.observe(sentinel.current!);
    return () => io.disconnect();
  }, []);

  const columns = useComputed(() => {
    const n = Math.max(1, Math.floor((width.value + 22) / 330));
    const colWidth = (width.value - 22 * (n - 1)) / n;
    const cols: Bookmark[][] = Array.from({ length: n }, () => []);
    const heights = new Array<number>(n).fill(0);
    for (const b of items.value) {
      const i = heights.indexOf(Math.min(...heights));
      cols[i]!.push(b);
      heights[i]! += estimate(b, colWidth) + 22;
    }
    return cols;
  });

  const v = view.value;
  const count = total.value;
  return (
    <section class="feed" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">{viewTitle(v)}</h1>
        <span class="feed-count">
          {loading.value && items.value.length === 0 ? "…" : `${count.toLocaleString()} ${count === 1 ? "post" : "posts"}`}
          {search.value && ` matching “${search.value}”`}
        </span>
      </div>
      {!loading.value && count === 0 && <Empty />}
      <div class="columns" ref={desk}>
        {columns.value.map((col, i) => (
          <div class="column" key={i}>
            {col.map((b) => (
              <Card key={b.tweet.id} b={b} />
            ))}
          </div>
        ))}
      </div>
      <div ref={sentinel} class="sentinel" aria-hidden="true" />
    </section>
  );
}

function Empty() {
  const libraryEmpty = (stats.value?.total ?? 0) === 0 && (stats.value?.removed ?? 0) === 0;
  if (!libraryEmpty) {
    return <p class="empty">Nothing here{search.value ? " matches that search" : " yet"}.</p>;
  }
  async function importFile() {
    try {
      const n = await Library.import();
      if (n) announce(`Imported ${n} bookmarks`);
      reload();
      refreshMeta();
    } catch (err) {
      fail(err);
    }
  }
  return (
    <div class="empty first-run">
      <p class="lede">The journal is empty.</p>
      <p>Sync to read your bookmarks from X. A window opens if X asks you to sign in.</p>
      <div class="row">
        <button class="press" disabled={isFixtureMode} onClick={() => startSync(true)}>
          Sync from X
        </button>
        <button class="press quiet" disabled={isFixtureMode} onClick={importFile}>
          Import an export…
        </button>
      </div>
    </div>
  );
}

function Sheet() {
  const id = opened.value;
  const b = find(id);
  const panel = useRef<HTMLElement>(null);
  const returnTo = useRef<HTMLElement | null>(null);
  const note = useSignal("");
  const reading = useSignal<string | null>(null);
  const hl = usePostHighlights(b);
  const suggested = useSuggestions(b);

  useSignalEffect(() => {
    const o = opened.value;
    if (o) {
      returnTo.current = document.activeElement as HTMLElement;
      requestAnimationFrame(() => panel.current?.focus());
    } else {
      returnTo.current?.focus();
    }
  });
  useEffect(() => {
    note.value = b?.note ?? "";
    reading.value = null;
    if (b) markOpened(b);
  }, [b?.tweet.id]);

  if (!b) return null;
  const t = b.tweet;
  if (reading.value) {
    return (
      <aside class="sheet wide" ref={panel} tabIndex={-1} aria-label="Saved page">
        <Reader
          url={reading.value}
          tweetId={t.id}
          quotes={hl.list.value.filter((h) => h.source === reading.value).map((h) => h.quote)}
          onAdd={hl.add}
          onClose={() => (reading.value = null)}
        />
      </aside>
    );
  }
  return (
    <aside class="sheet" ref={panel} tabIndex={-1} aria-label={`Post by ${t.authorName}`}>
      <div class="sheet-head">
        <span class="sheet-kicker">Kept post</span>
        <button class="close" onClick={() => (opened.value = null)} aria-label="Close">
          ✕
        </button>
      </div>
      <div class="byline big">
        <Avatar src={t.authorAvatar} size={40} />
        <span class="who">
          <span class="name">{t.authorName}</span>
          <span class="handle">@{t.authorHandle}</span>
        </span>
      </div>
      {isFixtureMode ? (
        <>
          {t.text && <Text tweet={t} />}
          <MediaGrid media={t.media} playable />
          {t.quoted && <Quoted tweet={t.quoted} />}
          <Thread b={b} />
        </>
      ) : (
        <>
          <Highlighter tweetId={t.id} onAdd={hl.add}>
            {t.text && <Text tweet={t} />}
            <MediaGrid media={t.media} playable />
            {t.quoted && <Quoted tweet={t.quoted} />}
            <Thread b={b} />
          </Highlighter>
          <HighlightList list={hl.list.value.filter((h) => !h.source)} onDelete={hl.remove} />
          <Links b={b} onRead={(url) => (reading.value = url)} />
        </>
      )}
      <p class="meta">
        Posted <time dateTime={t.createdAt}>{new Date(t.createdAt).toLocaleString()}</time>
        <br />
        In roost since <time dateTime={b.firstSeenAt}>{new Date(b.firstSeenAt).toLocaleDateString()}</time>
        {b.removedAt && (
          <>
            <br />
            Gone from X since <time dateTime={b.removedAt}>{new Date(b.removedAt).toLocaleDateString()}</time>
          </>
        )}
      </p>
      <a class="press link" href={t.url} onClick={(e) => openExternal(e, t.url)}>
        Open on X ↗
      </a>

      <fieldset class="file-under">
        <legend>File under</legend>
        <SuggestedChips ids={suggested.value} onFile={(id) => toggleCollection(b, id)} />
        {manual().length === 0 && <p class="hint">Make a collection in the sidebar to file posts.</p>}
        {manual().map((c, i) => (
          <label key={c.id} class="check" style={{ "--hue": ink(c.hue) }}>
            <input type="checkbox" disabled={isFixtureMode} checked={b.collections.includes(c.id)} onChange={() => toggleCollection(b, c.id)} />
            <span>{c.name}</span>
            {i < 9 && <kbd aria-hidden="true">{i + 1}</kbd>}
          </label>
        ))}
      </fieldset>

      <label class="note" for="note">
        <span class="legend">Field note</span>
        <textarea
          id="note"
          rows={4}
          value={note.value}
          disabled={isFixtureMode}
          onInput={(e) => {
            note.value = e.currentTarget.value;
            clearTimeout(noteTimer);
            noteTimer = setTimeout(() => saveNote(b, note.value), 500);
          }}
          onBlur={() => (clearTimeout(noteTimer), saveNote(b, note.value))}
          placeholder="Why you kept it…"
        />
      </label>
    </aside>
  );
}

let noteTimer: ReturnType<typeof setTimeout> | undefined;

function typing(e: KeyboardEvent) {
  const el = e.target as HTMLElement;
  return el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName);
}

function useKeyboard() {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "Escape" && opened.value) {
        opened.value = null;
        return;
      }
      if (typing(e)) return;
      if (view.value.kind === "sort") return;
      const list = items.value;
      const at = list.findIndex((b) => b.tweet.id === cursor.value);
      const current = at >= 0 ? list[at] : undefined;
      if (e.key === "/") {
        e.preventDefault();
        document.getElementById("search")?.focus();
      } else if (e.key === "j" || e.key === "k") {
        const next = list[Math.min(list.length - 1, Math.max(0, at + (e.key === "j" ? 1 : -1)))];
        if (next) {
          cursor.value = next.tweet.id;
          if (opened.value) opened.value = next.tweet.id;
          if (at >= list.length - 5) loadMore();
        }
      } else if (e.key === "Enter" && current) {
        opened.value = current.tweet.id;
      } else if (e.key === "o" && current) {
        Library.open(current.tweet.url);
      } else if (/^[1-9]$/.test(e.key) && current) {
        const c = manual()[Number(e.key) - 1];
        if (c) toggleCollection(current, c.id);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
}
