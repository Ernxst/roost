import { batch, effect, signal } from "@preact/signals";
import { events, Library, type Author, type Bookmark, type Collection, type Progress, type Highlight, type Query, type Revisit, type Stats } from "./mygo";
import { fixtureMode, isFixtureMode } from "./fixture-mode";

export type View =
  | { kind: "all" }
  | { kind: "unsorted" }
  | { kind: "removed" }
  | { kind: "collection"; id: number }
  | { kind: "author"; handle: string }
  | { kind: "sort" }
  | { kind: "revisit" }
  | { kind: "highlights" }
  | { kind: "wall" }
  | { kind: "duplicates" };

const PAGE = 60;

export const view = signal<View>({ kind: "all" });
export const search = signal("");
export const mediaOnly = signal(false);

export const items = signal<Bookmark[]>([]);
export const total = signal(0);
export const loading = signal(true);
export const stats = signal<Stats | null>(null);
export const collections = signal<Collection[]>([]);
export const authors = signal<Author[]>([]);

/** The bookmark the keyboard cursor is on. */
export const cursor = signal<string | null>(null);
/** The bookmark shown in the detail sheet. */
export const opened = signal<string | null>(null);
export const progress = signal<Progress | null>(null);
/** Text for the polite live region: results, sync outcomes, errors. */
export const announcement = signal("");
/**
 * The same message as a slip at the foot of the window. Results fade after a
 * few seconds; an error stays until it is dismissed or replaced.
 */
export const notice = signal<{ text: string; error: boolean; shown: boolean }>({ text: "", error: false, shown: false });
let noticeTimer: ReturnType<typeof setTimeout> | undefined;

export function announce(text: string, error = false) {
  announcement.value = "";
  requestAnimationFrame(() => (announcement.value = text));
  clearTimeout(noticeTimer);
  notice.value = { text, error, shown: true };
  if (!error) noticeTimer = setTimeout(dismissNotice, 4000);
}

export function dismissNotice() {
  clearTimeout(noticeTimer);
  notice.value = { ...notice.value, shown: false };
}

/** A number and the noun it counts, such as "1 note" or "1,204 notes". */
export function count(n: number, one: string, many = `${one}s`) {
  return `${n.toLocaleString()} ${n === 1 ? one : many}`;
}

/** An error from Go or the page, as a sentence. */
export function errorText(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  return message.charAt(0).toUpperCase() + message.slice(1);
}

export function fail(err: unknown) {
  announce(errorText(err), true);
}

function query(offset: number): Query {
  const v = view.value;
  return {
    // The sorting queue is unsorted posts not set aside.
    search: v.kind === "sort" ? `-is:aside ${search.value}` : search.value,
    mediaOnly: mediaOnly.value,
    collection: v.kind === "collection" ? v.id : v.kind === "unsorted" || v.kind === "sort" ? -1 : 0,
    author: v.kind === "author" ? v.handle : "",
    removed: v.kind === "removed",
    offset,
    limit: PAGE,
  };
}

let generation = 0;
let fixtureData: Promise<typeof import("./fixtures")> | undefined;

function loadFixtureData() {
  return (fixtureData ??= import("./fixtures"));
}

export async function reload() {
  if (view.value.kind === "revisit") return loadRevisit();
  if (view.value.kind === "highlights") return loadHighlights();
  // The wall and duplicates load their own bookmarks.
  if (view.value.kind === "wall" || view.value.kind === "duplicates") return;
  const g = ++generation;
  loading.value = true;
  if (import.meta.env.DEV && isFixtureMode) {
    try {
      const { fixtureList } = await loadFixtureData();
      const res = fixtureList(0, fixtureMode === "thousand" ? 1000 : PAGE);
      if (g !== generation) return;
      batch(() => {
        items.value = res.items;
        total.value = res.total;
      });
    } catch (err) {
      fail(err);
    } finally {
      if (g === generation) loading.value = false;
    }
    return;
  }
  try {
    const res = await Library.list(query(0));
    if (g !== generation) return;
    batch(() => {
      items.value = res.items;
      total.value = res.total;
    });
  } catch (err) {
    fail(err);
  } finally {
    if (g === generation) loading.value = false;
  }
}

export async function loadMore() {
  if (loading.value || items.value.length >= total.value) return;
  const g = generation;
  loading.value = true;
  if (import.meta.env.DEV && isFixtureMode) {
    try {
      const { fixtureList } = await loadFixtureData();
      const res = fixtureList(items.value.length, PAGE);
      if (g !== generation) return;
      batch(() => {
        items.value = [...items.value, ...res.items];
        total.value = res.total;
      });
    } catch (err) {
      fail(err);
    } finally {
      if (g === generation) loading.value = false;
    }
    return;
  }
  try {
    const res = await Library.list(query(items.value.length));
    if (g !== generation) return;
    const seen = new Set(items.value.map((b) => b.tweet.id));
    batch(() => {
      items.value = [...items.value, ...res.items.filter((b) => !seen.has(b.tweet.id))];
      total.value = res.total;
    });
  } catch (err) {
    fail(err);
  } finally {
    if (g === generation) loading.value = false;
  }
}

export async function refreshMeta() {
  if (import.meta.env.DEV && isFixtureMode) {
    try {
      const { fixtureMeta } = await loadFixtureData();
      const meta = fixtureMeta();
      batch(() => {
        stats.value = meta.stats;
        collections.value = meta.collections;
        authors.value = meta.authors;
      });
    } catch (err) {
      fail(err);
    }
    return;
  }
  try {
    const [s, c, a] = await Promise.all([Library.stats(), Library.collections(), Library.authors(8)]);
    batch(() => {
      stats.value = s;
      collections.value = c;
      authors.value = a;
    });
  } catch (err) {
    fail(err);
  }
}

// Reload whenever the view or filters change; typing waits for a pause.
let searchTimer: ReturnType<typeof setTimeout> | undefined;
let lastSearch = search.peek();
effect(() => {
  view.value;
  mediaOnly.value;
  const s = search.value;
  clearTimeout(searchTimer);
  if (s !== lastSearch) {
    lastSearch = s;
    searchTimer = setTimeout(reload, 140);
  } else {
    reload();
  }
});

/** The shelves of the Revisit view. */
export const revisit = signal<Revisit | null>(null);

export async function loadRevisit() {
  if (isFixtureMode) {
    revisit.value = { onThisDay: [], unopened: [], random: [] };
    return;
  }
  try {
    revisit.value = await Library.revisit(6);
  } catch (err) {
    fail(err);
  }
}

/** Bookmarks opened from outside the feed, such as from a highlight. */
const extra = signal<Bookmark[]>([]);

/** Opens a post in the sheet, loading it first if no list holds it. */
export async function openPost(id: string) {
  if (!find(id)) {
    try {
      const b = await Library.get(id);
      extra.value = [...extra.value.filter((x) => x.tweet.id !== id), b];
    } catch (err) {
      fail(err);
      return;
    }
  }
  opened.value = id;
}

export const highlights = signal<Highlight[]>([]);

export async function loadHighlights() {
  try {
    highlights.value = await Library.allHighlights();
  } catch (err) {
    fail(err);
  }
}

/** Finds a loaded bookmark, whether it is in the feed or on a Revisit shelf. */
export function find(id: string | null): Bookmark | undefined {
  if (!id) return undefined;
  const r = revisit.value;
  const all = [items.value, r?.onThisDay ?? [], r?.unopened ?? [], r?.random ?? [], extra.value];
  for (const list of all) {
    const b = list.find((x) => x.tweet.id === id);
    if (b) return b;
  }
}

function patch(id: string, fn: (b: Bookmark) => Bookmark) {
  const each = (list: Bookmark[]) => list.map((b) => (b.tweet.id === id ? fn(b) : b));
  batch(() => {
    items.value = each(items.value);
    extra.value = each(extra.value);
    const r = revisit.value;
    if (r) revisit.value = { onThisDay: each(r.onThisDay), unopened: each(r.unopened), random: each(r.random) };
  });
}

export function markOpened(b: Bookmark) {
  if (isFixtureMode) return;
  if (b.openedAt) return;
  patch(b.tweet.id, (x) => ({ ...x, openedAt: new Date().toISOString() }));
  Library.markOpened(b.tweet.id).catch((err) => fail(err));
}

export async function setAside(b: Bookmark, aside: boolean) {
  if (isFixtureMode) return;
  patch(b.tweet.id, (x) => ({ ...x, asideAt: aside ? new Date().toISOString() : null }));
  await Library.setAside(b.tweet.id, aside);
  refreshMeta();
}

/** Files a post into a collection, or takes it out, without the toggle's guesswork. */
export async function file(b: Bookmark, collectionID: number, inside: boolean) {
  if (isFixtureMode) return;
  patch(b.tweet.id, (x) => ({
    ...x,
    collections: inside
      ? [...new Set([...x.collections, collectionID])]
      : x.collections.filter((c) => c !== collectionID),
  }));
  await Library.setInCollection(b.tweet.id, collectionID, inside);
  refreshMeta();
}

export async function toggleCollection(b: Bookmark, collectionID: number) {
  if (isFixtureMode) return;
  const inside = !b.collections.includes(collectionID);
  patch(b.tweet.id, (x) => ({
    ...x,
    collections: inside ? [...x.collections, collectionID] : x.collections.filter((c) => c !== collectionID),
  }));
  const name = collections.value.find((c) => c.id === collectionID)?.name ?? "collection";
  try {
    await Library.setInCollection(b.tweet.id, collectionID, inside);
    announce(inside ? `Added to ${name}` : `Removed from ${name}`);
  } catch (err) {
    fail(err);
    reload();
  }
  refreshMeta();
}

export async function saveNote(b: Bookmark, note: string) {
  if (isFixtureMode) return;
  if (note.trim() === b.note) return;
  patch(b.tweet.id, (x) => ({ ...x, note: note.trim() }));
  try {
    await Library.setNote(b.tweet.id, note);
    announce("Note saved");
  } catch (err) {
    fail(err);
  }
}

// Hues for new collections, spaced so neighbours stay distinct.
const HUES = [28, 145, 250, 75, 320, 195, 5, 110, 280, 50];

/** Collections posts can be filed into, in sidebar order. */
export const manual = () => collections.value.filter((c) => !c.query);

export async function createCollection(name: string, query = "") {
  if (isFixtureMode) throw new Error("Collections are read-only in fixture mode");
  const hue = HUES[collections.value.length % HUES.length] ?? 28;
  const c = await Library.createCollection(name, hue, query);
  await refreshMeta();
  announce(`Created ${c.name}`);
  return c;
}

export async function startSync(full: boolean) {
  if (isFixtureMode) return;
  try {
    progress.value = { phase: "loading", full, seen: 0, added: 0, removed: 0, message: "" };
    await Library.sync(full);
  } catch (err) {
    progress.value = null;
    fail(err);
  }
}

let syncRefresh: ReturnType<typeof setTimeout> | undefined;
events.syncProgress.on((p) => {
  progress.value = p;
  if (p.phase === "reading") {
    clearTimeout(syncRefresh);
    syncRefresh = setTimeout(() => (reload(), refreshMeta()), 600);
  }
  if (p.phase === "done" || p.phase === "failed") {
    clearTimeout(syncRefresh);
    reload();
    refreshMeta();
    if (p.phase === "failed") announce(`Sync failed: ${p.message}`);
    else {
      const parts = [`${p.added} new`];
      if (p.removed) parts.push(`${p.removed} removed on X`);
      announce(`Sync finished: ${parts.join(", ")}`);
    }
  }
});

events.threadsFound.on((n) => {
  reload();
  announce(`Found ${count(n, "thread")} among your bookmarks`);
});

refreshMeta();
