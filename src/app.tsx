import { signal, useComputed, useSignal, useSignalEffect } from "@preact/signals";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";
import { Library, type Bookmark, type Collection } from "./mygo";
import { fixtureHref, fixtureMode, isFixtureMode, type FixtureMode } from "./fixture-mode";
import {
  announce,
  announcement,
  authors,
  collections,
  count,
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
  syncing,
  reload,
  refreshMeta,
  saveNote,
  search,
  selected,
  setAsideSelected,
  fileSelected,
  toggleSelected,
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
import { HighlightList, Highlighter, HighlightsView, marked, usePostHighlights } from "./highlights";
import { DuplicatesView, WallView } from "./wall";
import { WhatsNew } from "./changelog";
import { CollectionEditor } from "./collection";
import { inDialog, Keys } from "./keys";
import { SearchBox } from "./search";
import { author, AuthorHead, AuthorsView, showAuthor } from "./authors";
import { Avatar, longDate, longDay, MediaGrid, openExternal, previewRatio, Quoted, relative, Text } from "./tweet";

const same = (a: View, b: View) => JSON.stringify(a) === JSON.stringify(b);

export function App() {
  const lastFocused = useRef<HTMLElement | null>(null);
  useKeyboard();
  useEffect(() => {
    function rememberFocus() {
      if (!opened.value && document.activeElement instanceof HTMLElement) lastFocused.current = document.activeElement;
    }
    document.addEventListener("focusin", rememberFocus);
    return () => document.removeEventListener("focusin", rememberFocus);
  }, []);
  return (
    <div class="shell">
      <a class="skip" href="#feed" inert={!!opened.value}>
        Skip to bookmarks
      </a>
      <Rail inert={!!opened.value} />
      <main class="desk" id="feed" tabIndex={-1} inert={!!opened.value}>
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
        ) : view.value.kind === "authors" ? (
          <AuthorsView />
        ) : (
          <Feed />
        )}
      </main>
      <Sheet lastFocused={lastFocused} />
      <div class="sr-only" role="status" aria-live="polite" inert={!!opened.value}>
        {announcement.value}
      </div>
      <Notice inert={!!opened.value} />
      {import.meta.env.DEV && <FixtureSwitcher inert={!!opened.value} />}
    </div>
  );
}

/** The live region's message, shown for sighted readers too. */
function Notice({ inert }: { inert: boolean }) {
  const n = notice.value;
  return (
    <div class={n.error ? "notice error" : n.action ? "notice acting" : "notice"} data-shown={n.shown || undefined} aria-hidden={!n.shown || undefined} inert={inert}>
      <p>{n.text}</p>
      {n.action && (
        <button
          class="notice-action"
          tabIndex={n.shown ? 0 : -1}
          onClick={() => {
            dismissNotice();
            n.action?.run();
          }}
        >
          {n.action.label}
        </button>
      )}
      {(n.error || n.action) && (
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

function FixtureSwitcher({ inert }: { inert: boolean }) {
  return (
    <nav class="fixture-switcher" aria-label="Development data fixtures" inert={inert}>
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
        {smart && <span class="sr-only">, smart collection</span>}
        {count !== undefined && <span class="count">{count.toLocaleString()}</span>}
      </button>
    </li>
  );
}

function Rail({ inert }: { inert: boolean }) {
  const s = stats.value;
  const adding = useSignal(false);
  const name = useSignal("");
  const error = useSignal("");
  const nameField = useRef<HTMLInputElement>(null);
  // Focus moves only when the field opens, never on a re-render such as a navigation.
  useEffect(() => {
    if (adding.value) nameField.current?.focus();
  }, [adding.value]);

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
    <nav class="rail" aria-label="Library" inert={inert}>
      <div class="masthead">
        <span class="wordmark">roost</span>
        <span class="tagline">a field journal of the posts you kept</span>
      </div>

      <ul class="rail-list">
        <RailItem target={{ kind: "all" }} label="Everything" count={s?.total} />
        <RailItem target={{ kind: "unsorted" }} label="Unsorted" count={s?.unsorted} disabled={isFixtureMode} />
        {!!s?.removed && <RailItem target={{ kind: "removed" }} label="Missing from bookmarks" count={s.removed} disabled={isFixtureMode} />}
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
            onBlur={() => !name.value.trim() && (adding.value = false)}
            placeholder="Name it…"
            autoComplete="off"
            aria-invalid={error.value ? "true" : undefined}
            aria-describedby={error.value ? "new-collection-error" : undefined}
            ref={nameField}
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
                  <Avatar src={a.avatar} name={a.name || a.handle} size={18} />
                  <span class="label">{a.name || a.handle}</span>
                  <span class="count">{a.count.toLocaleString()}</span>
                </button>
              </li>
            ))}
            <RailItem target={{ kind: "authors" }} label="All authors" disabled={isFixtureMode} />
          </ul>
        </>
      )}
      <div class="rail-foot">
        {!isFixtureMode && <SettingsButton />}
        <Keys />
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
  if (p?.phase === "failed") {
    return (
      <span class="ticker failed">
        <span class="ticker-mark" aria-hidden="true">!</span>
        Sync failed
      </span>
    );
  }
  if (!last) return <span class="ticker">Never synced</span>;
  return (
    <span class="ticker" title={longDate(last.finishedAt)}>
      Synced {relative(last.finishedAt)} · {last.added.toLocaleString()} new
      {last.removed ? ` · ${last.removed.toLocaleString()} missing from bookmarks` : ""}
    </span>
  );
}

const dismissedSyncNotice = signal<unknown>(null);

/** The full text of a failed or incomplete sync. It floats under the toolbar so nothing below it moves. */
function SyncNotice() {
  const p = progress.value;
  const failed = p?.phase === "failed";
  if (!p || !p.message || (!failed && p.phase !== "done") || dismissedSyncNotice.value === p) return null;
  return (
    <div class={failed ? "sync-notice failed" : "sync-notice"} role={failed ? "alert" : "status"}>
      <p>{failed ? `Sync failed: ${p.message}` : p.message}</p>
      <button class="press quiet" onClick={() => (dismissedSyncNotice.value = p)}>
        Dismiss
      </button>
    </div>
  );
}

function Toolbar() {
  const showFeedFilters = ["all", "unsorted", "removed", "collection", "author", "sort"].includes(view.value.kind);
  return (
    <header class="toolbar">
      {showFeedFilters && (
        <>
          <SearchBox />
          {search.value.trim() && !isFixtureMode && <SaveSearch />}
          <button class="chip" disabled={isFixtureMode} aria-pressed={mediaOnly.value} onClick={() => (mediaOnly.value = !mediaOnly.value)}>
            With media
          </button>
        </>
      )}
      <div class="sync">
        <SyncStatus />
        <button class="press" disabled={syncing.value || isFixtureMode} onClick={() => startSync(false)}>
          Sync
        </button>
      </div>
      <SyncNotice />
    </header>
  );
}

function SaveSearch() {
  const naming = useSignal(false);
  const name = useSignal("");
  const error = useSignal("");
  const nameField = useRef<HTMLInputElement>(null);
  // Focus moves only when the field opens, never on a re-render such as a navigation.
  useEffect(() => {
    if (naming.value) nameField.current?.focus();
  }, [naming.value]);
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
        onBlur={() => !name.value.trim() && (naming.value = false)}
        placeholder="Name this search…"
        autoComplete="off"
        aria-invalid={error.value ? "true" : undefined}
        aria-describedby={error.value ? "save-search-error" : undefined}
        ref={nameField}
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
      return "Missing from bookmarks";
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
    case "authors":
      return "Authors";
    case "author": {
      const a = author.value?.handle.toLowerCase() === v.handle.toLowerCase() ? author.value : authors.value.find((a) => a.handle === v.handle);
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
  const editing = useSignal<number | null>(null);
  const editButton = useRef<HTMLButtonElement>(null);

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
  const shown = v.kind === "collection" ? collections.value.find((c) => c.id === v.id) : undefined;
  return (
    <section class="feed" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">{viewTitle(v)}</h1>
        <span class="feed-count">
          {loading.value && items.value.length === 0 ? "Loading…" : count(total.value, "post")}
          {search.value && ` matching “${search.value}”`}
        </span>
        {shown && !isFixtureMode && <ShareCollection id={shown.id} />}
        {shown && !isFixtureMode && (
          <button
            ref={editButton}
            class="chip edit"
            aria-expanded={editing.value === shown.id}
            onClick={() => (editing.value = editing.value === shown.id ? null : shown.id)}
          >
            Edit collection
          </button>
        )}
      </div>
      {shown && editing.value === shown.id && (
        <CollectionEditor
          key={shown.id}
          c={shown}
          onClose={() => {
            editing.value = null;
            editButton.current?.focus();
          }}
        />
      )}
      {v.kind === "author" && !isFixtureMode && <AuthorHead handle={v.handle} />}
      {!loading.value && total.value === 0 && <Empty />}
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
      <SelectionBar />
    </section>
  );
}

function emptyText(v: View): string {
  switch (v.kind) {
    case "unsorted":
      return "Every post is filed.";
    case "removed":
      return "Every kept post is still on X.";
    case "collection":
      return collections.value.find((c) => c.id === v.id)?.query
        ? "No posts match this collection’s search yet."
        : "Nothing filed here yet. Open a post and file it under this collection, or sort the unsorted.";
    default:
      return "Nothing here yet.";
  }
}

/** Actions on the selected bookmarks, shown while any are selected. */
/** Saves the collection as one Markdown file to send to someone. */
function ShareCollection({ id }: { id: number }) {
  const saving = useSignal(false);
  const sharing = useSignal(false);
  const includeNotes = useSignal(false);
  async function share() {
    saving.value = true;
    try {
      const path = await Library.shareCollection(id, includeNotes.value);
      if (path) {
        announce(`Saved ${path.split("/").pop()}`);
        sharing.value = false;
      }
    } catch (err) {
      fail(err);
    } finally {
      saving.value = false;
    }
  }
  return (
    <div class="share">
      <button
        class="chip"
        type="button"
        aria-expanded={sharing.value}
        aria-controls="share-options"
        onClick={() => {
          includeNotes.value = false;
          sharing.value = !sharing.value;
        }}
      >
        Share as Markdown…
      </button>
      {sharing.value && (
        <form
          id="share-options"
          class="share-options"
          aria-label="Share options"
          onSubmit={(e) => {
            e.preventDefault();
            share();
          }}
          onKeyDown={(e) => e.key === "Escape" && (sharing.value = false)}
        >
          <label for="include-private-notes">
            <input
              id="include-private-notes"
              type="checkbox"
              checked={includeNotes.value}
              onChange={(e) => (includeNotes.value = e.currentTarget.checked)}
            />
            Include private notes and highlight notes
          </label>
          <div class="share-actions">
            <button class="press quiet" type="button" disabled={saving.value} onClick={() => (sharing.value = false)}>
              Cancel
            </button>
            <button class="press" type="submit" disabled={saving.value}>
              {saving.value ? "Saving…" : "Save Markdown"}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}

function SelectionBar() {
  const target = useSignal("");
  const n = selected.value.size;
  if (n === 0) return null;
  const v = view.value;
  return (
    <form
      class="selection-bar"
      aria-label="Selected posts"
      onSubmit={(e) => {
        e.preventDefault();
        if (target.value) fileSelected(Number(target.value));
      }}
    >
      <span class="selection-count">
        {n.toLocaleString()} selected
      </span>
      {manual().length > 0 && (
        <>
          <label for="file-selected" class="sr-only">
            Collection
          </label>
          <select id="file-selected" value={target.value} onChange={(e) => (target.value = e.currentTarget.value)}>
            <option value="">Choose a collection…</option>
            {manual().map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <button class="press" type="submit" disabled={!target.value}>
            File
          </button>
        </>
      )}
      {v.kind === "unsorted" && (
        <button class="press quiet" type="button" onClick={setAsideSelected} title="Leave them out of the sorting queue">
          Set aside
        </button>
      )}
      <button class="press quiet" type="button" onClick={() => (selected.value = new Set(items.value.map((b) => b.tweet.id)))}>
        Select all {items.value.length.toLocaleString()}
      </button>
      <button class="press quiet" type="button" onClick={() => (selected.value = new Set())}>
        <kbd aria-hidden="true">esc</kbd>Clear
      </button>
    </form>
  );
}

function Empty() {
  const libraryEmpty = (stats.value?.total ?? 0) === 0 && (stats.value?.removed ?? 0) === 0;
  if (!libraryEmpty && (search.value || mediaOnly.value)) {
    return (
      <div class="empty">
        <p>{search.value ? `No posts match “${search.value}”${mediaOnly.value ? " among posts with media" : ""}.` : "No posts with media here."}</p>
        <button
          class="press quiet"
          onClick={() => {
            search.value = "";
            mediaOnly.value = false;
          }}
        >
          {search.value ? "Clear search" : "Show every post"}
        </button>
      </div>
    );
  }
  if (!libraryEmpty) return <p class="empty">{emptyText(view.value)}</p>;
  async function importFile() {
    try {
      const n = await Library.import();
      if (n) announce(`Imported ${count(n, "bookmark")}`);
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

function Sheet({ lastFocused }: { lastFocused: { current: HTMLElement | null } }) {
  const id = opened.value;
  const b = find(id);
  const panel = useRef<HTMLDivElement>(null);
  const returnTo = useRef<HTMLElement | null>(null);
  const wasOpen = useRef(false);
  const note = useSignal("");
  const reading = useSignal<string | null>(null);
  const hl = usePostHighlights(b);
  const quotes = hl.list.value.filter((h) => !h.source).map((h) => h.quote);
  const suggested = useSuggestions(b);

  useSignalEffect(() => {
    const o = opened.value;
    if (o) {
      if (!wasOpen.current) returnTo.current = lastFocused.current ?? (document.activeElement as HTMLElement);
      wasOpen.current = true;
      requestAnimationFrame(() => panel.current?.focus());
    } else if (wasOpen.current) {
      wasOpen.current = false;
      const target = returnTo.current;
      returnTo.current = null;
      if (target?.isConnected) target.focus();
    }
  });
  useEffect(() => {
    function trapFocus(e: KeyboardEvent) {
      if (!opened.value || e.key !== "Tab" || !panel.current) return;
      const focusable = [...panel.current.querySelectorAll<HTMLElement>(
        'a[href], button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])',
      )].filter((el) => el.getClientRects().length > 0);
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (!first || !last) {
        e.preventDefault();
        panel.current.focus();
      } else if (e.shiftKey && (document.activeElement === first || !panel.current.contains(document.activeElement))) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && (document.activeElement === last || !panel.current.contains(document.activeElement))) {
        e.preventDefault();
        first.focus();
      }
    }
    window.addEventListener("keydown", trapFocus);
    return () => window.removeEventListener("keydown", trapFocus);
  }, []);
  useEffect(() => {
    note.value = b?.note ?? "";
    reading.value = null;
    if (b) markOpened(b);
  }, [b?.tweet.id]);

  if (!b) return null;
  const t = b.tweet;
  if (reading.value) {
    return (
      <div class="sheet wide" ref={panel} tabIndex={-1} role="dialog" aria-modal="true" aria-label="Saved page">
        <Reader
          url={reading.value}
          tweetId={t.id}
          quotes={hl.list.value.filter((h) => h.source === reading.value).map((h) => h.quote)}
          onAdd={hl.add}
          onClose={() => (reading.value = null)}
        />
      </div>
    );
  }
  return (
    <div class="sheet" ref={panel} tabIndex={-1} role="dialog" aria-modal="true" aria-label={`Post by ${t.authorName}`}>
      <div class="sheet-head">
        <span class="sheet-kicker">Kept post</span>
        <button class="close" onClick={() => (opened.value = null)} aria-label="Close">
          ✕
        </button>
      </div>
      <div class="byline big">
        <Avatar src={t.authorAvatar} name={t.authorName} size={40} />
        <h2 class="who">
          <button
            class="author-link"
            disabled={isFixtureMode}
            onClick={() => {
              opened.value = null;
              showAuthor(t.authorHandle);
            }}
            title={`Every post kept by @${t.authorHandle}`}
          >
            <span class="name" dir="auto">{t.authorName}</span>
            <span class="handle">@{t.authorHandle}</span>
          </button>
        </h2>
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
            {t.text && <Text tweet={t} mark={(s) => marked(s, quotes)} />}
            <MediaGrid media={t.media} playable />
            {t.quoted && <Quoted tweet={t.quoted} />}
            <Thread b={b} />
          </Highlighter>
          <Links b={b} onRead={(url) => (reading.value = url)} />
        </>
      )}
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
          dir="auto"
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
      {!isFixtureMode && <HighlightList list={hl.list.value.filter((h) => !h.source)} onDelete={hl.remove} onNote={hl.note} />}
      <p class="meta">
        Posted <time dateTime={t.createdAt}>{longDate(t.createdAt)}</time>
        <br />
        In roost since <time dateTime={b.firstSeenAt}>{longDay(b.firstSeenAt)}</time>
        {b.removedAt && (
          <>
            <br />
            Missing from bookmarks since <time dateTime={b.removedAt}>{longDay(b.removedAt)}</time>
          </>
        )}
      </p>
      <a class="press link" href={t.url} onClick={(e) => openExternal(e, t.url)}>
        Open on X ↗
      </a>
    </div>
  );
}

let noteTimer: ReturnType<typeof setTimeout> | undefined;

function typing(e: KeyboardEvent) {
  const el = e.target as HTMLElement;
  return (
    el.isContentEditable ||
    ["INPUT", "TEXTAREA", "SELECT", "BUTTON", "A", "SUMMARY"].includes(el.tagName) ||
    !!el.closest("[role='button'], [role='link']")
  );
}

function useKeyboard() {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.metaKey || e.ctrlKey || e.altKey || inDialog(e)) return;
      if (e.key === "Escape" && opened.value) {
        opened.value = null;
        return;
      }
      if (e.key === "Escape" && selected.value.size > 0 && !typing(e)) {
        selected.value = new Set();
        return;
      }
      if (typing(e)) return;
      // Search narrows the sorting queue too, so / works everywhere.
      if (e.key === "/") {
        e.preventDefault();
        document.getElementById("search")?.focus();
        return;
      }
      if (view.value.kind === "sort") return;
      const list = items.value;
      const at = list.findIndex((b) => b.tweet.id === cursor.value);
      const current = at >= 0 ? list[at] : undefined;
      if (e.key === "j" || e.key === "k") {
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
      } else if (e.key === "x" && current) {
        toggleSelected(current.tweet.id);
      } else if (/^[1-9]$/.test(e.key) && current) {
        const c = manual()[Number(e.key) - 1];
        if (c) toggleCollection(current, c.id);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
}
