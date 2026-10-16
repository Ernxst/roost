import { batch, effect, signal } from "@preact/signals";
import { events, Library, type Author, type Bookmark, type Collection, type Progress, type Query, type Revisit, type Stats } from "./mygo";

export type View =
  | { kind: "all" }
  | { kind: "unsorted" }
  | { kind: "removed" }
  | { kind: "collection"; id: number }
  | { kind: "author"; handle: string }
  | { kind: "sort" }
  | { kind: "revisit" };

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

export function announce(text: string) {
  announcement.value = "";
  requestAnimationFrame(() => (announcement.value = text));
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

export async function reload() {
  if (view.value.kind === "revisit") return loadRevisit();
  const g = ++generation;
  loading.value = true;
  try {
    const res = await Library.list(query(0));
    if (g !== generation) return;
    batch(() => {
      items.value = res.items;
      total.value = res.total;
    });
  } catch (err) {
    announce(String(err));
  } finally {
    if (g === generation) loading.value = false;
  }
}

export async function loadMore() {
  if (loading.value || items.value.length >= total.value) return;
  const g = generation;
  loading.value = true;
  try {
    const res = await Library.list(query(items.value.length));
    if (g !== generation) return;
    const seen = new Set(items.value.map((b) => b.tweet.id));
    batch(() => {
      items.value = [...items.value, ...res.items.filter((b) => !seen.has(b.tweet.id))];
      total.value = res.total;
    });
  } finally {
    if (g === generation) loading.value = false;
  }
}

export async function refreshMeta() {
  const [s, c, a] = await Promise.all([Library.stats(), Library.collections(), Library.authors(8)]);
  batch(() => {
    stats.value = s;
    collections.value = c;
    authors.value = a;
  });
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
  try {
    revisit.value = await Library.revisit(6);
  } catch (err) {
    announce(String(err));
  }
}

/** Finds a loaded bookmark, whether it is in the feed or on a Revisit shelf. */
export function find(id: string | null): Bookmark | undefined {
  if (!id) return undefined;
  const r = revisit.value;
  const all = [items.value, r?.onThisDay ?? [], r?.unopened ?? [], r?.random ?? []];
  for (const list of all) {
    const b = list.find((x) => x.tweet.id === id);
    if (b) return b;
  }
}

function patch(id: string, fn: (b: Bookmark) => Bookmark) {
  const each = (list: Bookmark[]) => list.map((b) => (b.tweet.id === id ? fn(b) : b));
  batch(() => {
    items.value = each(items.value);
    const r = revisit.value;
    if (r) revisit.value = { onThisDay: each(r.onThisDay), unopened: each(r.unopened), random: each(r.random) };
  });
}

export function markOpened(b: Bookmark) {
  if (b.openedAt) return;
  patch(b.tweet.id, (x) => ({ ...x, openedAt: new Date().toISOString() }));
  Library.markOpened(b.tweet.id).catch((err) => announce(String(err)));
}

export async function setAside(b: Bookmark, aside: boolean) {
  patch(b.tweet.id, (x) => ({ ...x, asideAt: aside ? new Date().toISOString() : null }));
  await Library.setAside(b.tweet.id, aside);
  events.threadsFound.on((n) => {
  reload();
  announce(`Found ${n} ${n === 1 ? "thread" : "threads"} among your bookmarks`);
});

refreshMeta();
}

/** Files a post into a collection, or takes it out, without the toggle's guesswork. */
export async function file(b: Bookmark, collectionID: number, inside: boolean) {
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
    announce(String(err));
    reload();
  }
  refreshMeta();
}

export async function saveNote(b: Bookmark, note: string) {
  if (note.trim() === b.note) return;
  patch(b.tweet.id, (x) => ({ ...x, note: note.trim() }));
  await Library.setNote(b.tweet.id, note);
  announce("Note saved");
}

// Hues for new collections, spaced so neighbours stay distinct.
const HUES = [28, 145, 250, 75, 320, 195, 5, 110, 280, 50];

/** Collections posts can be filed into, in sidebar order. */
export const manual = () => collections.value.filter((c) => !c.query);

export async function createCollection(name: string, query = "") {
  const hue = HUES[collections.value.length % HUES.length] ?? 28;
  const c = await Library.createCollection(name, hue, query);
  await refreshMeta();
  announce(`Created ${c.name}`);
  return c;
}

export async function startSync(full: boolean) {
  try {
    progress.value = { phase: "loading", full, seen: 0, added: 0, removed: 0, message: "" };
    await Library.sync(full);
  } catch (err) {
    progress.value = null;
    announce(String(err));
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

refreshMeta();
