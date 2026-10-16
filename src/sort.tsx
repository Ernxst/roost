import { useComputed, useSignal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";
import { ink, Stamps } from "./card";
import { inDialog } from "./keys";
import { Library, type Bookmark } from "./mygo";
import { announce, createCollection, fail, file, items, loading, manual, opened, reload, setAside, stats, total } from "./state";
import { useSuggestions } from "./suggest";
import { Avatar, MediaGrid, openExternal, Quoted, shortDate, Text } from "./tweet";

type Action = { b: Bookmark; kind: "file"; collection: number } | { b: Bookmark; kind: "aside" | "skip" };

/**
 * Sorting mode shows the unsorted posts one at a time and files each with a
 * single key, so a backlog can be cleared in one sitting.
 */
export function SortMode() {
  // Posts dealt with this session; the queue is whatever is left.
  const done = useSignal<Set<string>>(new Set());
  const history = useSignal<Action[]>([]);
  const naming = useSignal(false);
  const name = useSignal("");
  const nameField = useRef<HTMLInputElement>(null);
  const queue = useComputed(() => items.value.filter((b) => !done.value.has(b.tweet.id)));
  const current = queue.value[0];
  const suggested = useSuggestions(current);

  // Focus moves only when the field opens, never on a re-render such as dealing the next post.
  useEffect(() => {
    if (naming.value) nameField.current?.focus();
  }, [naming.value]);

  // Filed posts leave the unsorted list, so a fresh first page is the next batch.
  useEffect(() => {
    if (!loading.value && queue.value.length < 5 && items.value.length < total.value) reload();
  }, [queue.value.length]);

  function advance(a: Action) {
    history.value = [...history.value, a];
    done.value = new Set([...done.value, a.b.tweet.id]);
  }

  async function fileInto(index: number, stay: boolean) {
    const c = manual()[index];
    if (!current || !c) return;
    const inside = !(stay && current.collections.includes(c.id));
    try {
      await file(current, c.id, inside);
      announce(inside ? `Filed under ${c.name}` : `Taken out of ${c.name}`);
      if (!stay) advance({ b: current, kind: "file", collection: c.id });
    } catch (err) {
      fail(err);
    }
  }

  async function aside() {
    if (!current) return;
    try {
      await setAside(current, true);
      announce("Set aside");
      advance({ b: current, kind: "aside" });
    } catch (err) {
      fail(err);
    }
  }

  function skip() {
    if (!current) return;
    advance({ b: current, kind: "skip" });
  }

  async function undo() {
    const last = history.value.at(-1);
    if (!last) return;
    history.value = history.value.slice(0, -1);
    const next = new Set(done.value);
    next.delete(last.b.tweet.id);
    done.value = next;
    // Put the post back at the front of the queue.
    items.value = [last.b, ...items.value.filter((b) => b.tweet.id !== last.b.tweet.id)];
    try {
      if (last.kind === "file") await file(last.b, last.collection, false);
      if (last.kind === "aside") await setAside(last.b, false);
      announce("Undone");
    } catch (err) {
      fail(err);
    }
  }

  async function createAndFile(e: Event) {
    e.preventDefault();
    if (!current) return;
    try {
      const c = await createCollection(name.value);
      await file(current, c.id, true);
      advance({ b: current, kind: "file", collection: c.id });
      naming.value = false;
      name.value = "";
    } catch (err) {
      fail(err);
    }
  }

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.metaKey || e.ctrlKey || e.altKey || opened.value || inDialog(e)) return;
      const el = e.target as HTMLElement;
      if (["INPUT", "TEXTAREA"].includes(el.tagName)) return;
      const digit = /^Digit([1-9])$/.exec(e.code);
      if (digit) fileInto(Number(digit[1]) - 1, e.shiftKey);
      else if (e.key === "x") aside();
      else if (e.key === "s" || e.key === "ArrowRight") skip();
      else if (e.key === "u" || e.key === "Backspace") undo();
      else if (e.key === "n") naming.value = true;
      else if (e.key === "o" && current) Library.open(current.tweet.url);
      else if (e.key === "Enter" && current) opened.value = current.tweet.id;
      else return;
      e.preventDefault();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const left = stats.value?.toSort ?? 0;
  // Unsorted posts set aside wait outside the queue.
  const asideCount = (stats.value?.unsorted ?? 0) - left;
  const filed = history.value.filter((a) => a.kind === "file").length;
  return (
    <section class="sorter" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">Sorting</h1>
        <span class="feed-count">
          {left.toLocaleString()} left to sort{asideCount > 0 ? ` · ${asideCount.toLocaleString()} set aside` : ""}
          {filed ? ` · ${filed.toLocaleString()} filed this sitting` : ""}
        </span>
      </div>
      {!current ? (
        <div class="empty first-run">
          {loading.value ? <p class="lede">Dealing the next posts…</p> : <p class="lede dealt">Nothing left to sort.</p>}
          {!loading.value && done.value.size > 0 && (
            <button class="press quiet" onClick={() => ((done.value = new Set()), reload())}>
              Go round again
            </button>
          )}
        </div>
      ) : (
        <div class="sort-table">
          <SortCard b={current} />
          <div class="sort-keys">
            <p class="hints-title">File under</p>
            {manual().length === 0 && <p class="hint">No collections yet. Press n to make one and file this post in it.</p>}
            <ol class="bins">
              {manual().map((c, i) => (
                <li key={c.id}>
                  <button
                    class="bin"
                    style={{ "--hue": ink(c.hue) }}
                    aria-pressed={current.collections.includes(c.id)}
                    data-suggested={suggested.value.includes(c.id) || undefined}
                    onClick={(e) => fileInto(i, e.shiftKey)}
                  >
                    {i < 9 && <kbd>{i + 1}</kbd>}
                    <span>{c.name}</span>
                    {suggested.value[0] === c.id && <span class="bin-hint">likely</span>}
                    {current.collections.includes(c.id) && (
                      <span class="bin-check" aria-hidden="true">
                        ✓
                      </span>
                    )}
                  </button>
                </li>
              ))}
            </ol>
            {naming.value ? (
              <form class="new-collection" onSubmit={createAndFile}>
                <label for="sort-new" class="field-label">
                  New collection
                </label>
                <input
                  id="sort-new"
                  value={name.value}
                  onInput={(e) => (name.value = e.currentTarget.value)}
                  onKeyDown={(e) => e.key === "Escape" && (naming.value = false)}
                  placeholder="New collection…"
                  autoComplete="off"
                  ref={nameField}
                />
              </form>
            ) : (
              <button class="add" onClick={() => (naming.value = true)}>
                <kbd>n</kbd> New collection
              </button>
            )}
            <div class="sort-actions">
              <button class="press quiet" onClick={skip}>
                <kbd>s</kbd> Skip
              </button>
              <button class="press quiet" onClick={aside}>
                <kbd>x</kbd> Set aside
              </button>
              <button class="press quiet" onClick={undo} disabled={history.value.length === 0}>
                <kbd>u</kbd> Undo
              </button>
            </div>
            <p class="hint">Shift and a number files it without moving on, for posts that belong in more than one place.</p>
          </div>
        </div>
      )}
    </section>
  );
}

function SortCard({ b }: { b: Bookmark }) {
  const t = b.tweet;
  return (
    <article class="card big" aria-labelledby={`sort-${t.id}`} aria-live="polite">
      <header class="byline">
        <Avatar src={t.authorAvatar} name={t.authorName} size={36} />
        <span class="who" id={`sort-${t.id}`}>
          <span class="name" dir="auto">{t.authorName}</span>
          <span class="handle">@{t.authorHandle}</span>
        </span>
        <time dateTime={t.createdAt}>{shortDate(t.createdAt)}</time>
      </header>
      {t.text && <Text tweet={t} />}
      <MediaGrid media={t.media} playable />
      {t.quoted && <Quoted tweet={t.quoted} />}
      <Stamps b={b} />
      <a class="press link quiet" href={t.url} onClick={(e) => openExternal(e, t.url)}>
        <kbd>o</kbd> Open on X ↗
      </a>
    </article>
  );
}
