import { useSignal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";
import type { ComponentChildren } from "preact";
import { Library, type Bookmark, type Highlight } from "./mygo";
import { announce, fail, highlights, openPost } from "./state";
import { isFixtureMode } from "./fixture-mode";
import { Avatar, shortDate } from "./tweet";

/**
 * Wraps text the user can highlight. Selecting text inside shows a
 * Highlight button by the selection; h does the same from the keyboard.
 */
export function Highlighter({
  tweetId,
  source = "",
  onAdd,
  children,
}: {
  tweetId: string;
  source?: string;
  onAdd: (h: Highlight) => void;
  children: ComponentChildren;
}) {
  const box = useRef<HTMLDivElement>(null);
  const at = useSignal<{ x: number; y: number } | null>(null);

  function selected(): string {
    const sel = getSelection();
    if (!sel || sel.isCollapsed || !box.current) return "";
    const range = sel.getRangeAt(0);
    if (!box.current.contains(range.commonAncestorContainer)) return "";
    return sel.toString().trim();
  }

  function update() {
    const text = selected();
    if (!text) {
      at.value = null;
      return;
    }
    const rect = getSelection()!.getRangeAt(0).getBoundingClientRect();
    at.value = { x: rect.left + rect.width / 2, y: rect.top };
  }

  async function save() {
    const text = selected();
    if (!text) return;
    try {
      const h = await Library.addHighlight(tweetId, source, text);
      getSelection()?.removeAllRanges();
      at.value = null;
      onAdd(h);
      announce("Highlighted");
    } catch (err) {
      fail(err);
    }
  }

  useEffect(() => {
    document.addEventListener("selectionchange", update);
    function onKey(e: KeyboardEvent) {
      if (e.key === "h" && !e.metaKey && !e.ctrlKey && selected()) {
        e.preventDefault();
        save();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("selectionchange", update);
      window.removeEventListener("keydown", onKey);
    };
  }, [tweetId, source]);

  return (
    <div ref={box} class="highlightable">
      {children}
      {at.value && (
        <button
          class="highlight-pop"
          style={{ left: `${at.value.x}px`, top: `${at.value.y}px` }}
          onMouseDown={(e) => e.preventDefault()}
          onClick={save}
        >
          Highlight <kbd>h</kbd>
        </button>
      )}
    </div>
  );
}

/** Marks every highlighted passage in a paragraph of plain text. */
export function marked(text: string, quotes: string[]): ComponentChildren {
  const spans: [number, number][] = [];
  for (const q of quotes) {
    for (let i = text.indexOf(q); i >= 0 && q; i = text.indexOf(q, i + q.length)) spans.push([i, i + q.length]);
  }
  if (spans.length === 0) return text;
  spans.sort((a, b) => a[0] - b[0]);
  const out: ComponentChildren[] = [];
  let pos = 0;
  for (const [from, to] of spans) {
    if (from < pos) continue;
    out.push(text.slice(pos, from), <mark key={from}>{text.slice(from, to)}</mark>);
    pos = to;
  }
  out.push(text.slice(pos));
  return out;
}

/** A note on one highlight, saved when the user pauses or leaves the field. */
function HighlightNote({ h, onSave }: { h: Highlight; onSave: (id: number, note: string) => void }) {
  const note = useSignal(h.note);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => {
    note.value = h.note;
  }, [h.id]);
  function save() {
    clearTimeout(timer.current);
    if (note.value.trim() !== h.note) onSave(h.id, note.value.trim());
  }
  return (
    <>
      <label for={`hl-note-${h.id}`} class="sr-only">
        Note on this highlight
      </label>
      <textarea
        id={`hl-note-${h.id}`}
        class="highlight-note"
        dir="auto"
        rows={1}
        value={note.value}
        disabled={isFixtureMode}
        placeholder="Add a note…"
        onInput={(e) => {
          note.value = e.currentTarget.value;
          clearTimeout(timer.current);
          timer.current = setTimeout(save, 500);
        }}
        onBlur={save}
      />
    </>
  );
}

/** Saves a highlight's note and updates the list it came from. */
export async function saveHighlightNote(list: { value: Highlight[] }, id: number, note: string) {
  list.value = list.value.map((h) => (h.id === id ? { ...h, note } : h));
  try {
    await Library.setHighlightNote(id, note);
    announce("Note saved");
  } catch (err) {
    fail(err);
  }
}

/** A post's highlights, listed in the sheet. */
export function HighlightList({
  list,
  onDelete,
  onNote,
}: {
  list: Highlight[];
  onDelete: (id: number) => void;
  onNote: (id: number, note: string) => void;
}) {
  if (list.length === 0) return null;
  return (
    <section class="highlight-list" aria-label="Highlights">
      <h3 class="legend">Highlights</h3>
      <ul>
        {list.map((h) => (
          <li key={h.id}>
            <blockquote dir="auto">{h.quote}</blockquote>
            {h.source && <span class="link-status">from {new URL(h.source).hostname.replace(/^www\./, "")}</span>}
            <HighlightNote h={h} onSave={onNote} />
            <button class="close" aria-label="Remove highlight" onClick={() => onDelete(h.id)}>
              ✕
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}

/** Every highlight in the library, newest first. */
export function HighlightsView() {
  const list = highlights.value;
  return (
    <section class="highlights-view" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">Highlights</h1>
        <span class="feed-count">
          {list.length.toLocaleString()} {list.length === 1 ? "passage" : "passages"}
        </span>
      </div>
      {list.length === 0 ? (
        <p class="empty">Select text in a post or a saved page and press h to keep it here.</p>
      ) : (
        <ol class="clippings">
          {list.map((h) => (
            <li key={h.id} class="clipping">
              <blockquote dir="auto">{h.quote}</blockquote>
              <HighlightNote h={h} onSave={(id, note) => saveHighlightNote(highlights, id, note)} />
              <button class="clip-source" onClick={() => openPost(h.tweetId)}>
                {h.tweet && <Avatar src={h.tweet.authorAvatar} name={h.tweet.authorName} size={18} />}
                <span>
                  {h.tweet?.authorName}
                  {h.source && ` · ${new URL(h.source).hostname.replace(/^www\./, "")}`}
                </span>
                <time dateTime={h.createdAt}>{shortDate(h.createdAt)}</time>
              </button>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

/** Loads and edits the highlights of the post open in the sheet. */
export function usePostHighlights(b: Bookmark | undefined) {
  const list = useSignal<Highlight[]>([]);
  useEffect(() => {
    list.value = [];
    if (b) Library.highlights(b.tweet.id).then((h) => (list.value = h));
  }, [b?.tweet.id]);
  async function restore(h: Highlight) {
    try {
      const back = await Library.addHighlight(h.tweetId, h.source, h.quote);
      list.value = [...list.value, back];
      announce("Highlight restored");
    } catch (err) {
      fail(err);
    }
  }
  return {
    list,
    add: (h: Highlight) => (list.value = [...list.value, h]),
    note: (id: number, note: string) => saveHighlightNote(list, id, note),
    remove: async (id: number) => {
      const before = list.value;
      const gone = before.find((h) => h.id === id);
      list.value = before.filter((h) => h.id !== id);
      try {
        await Library.deleteHighlight(id);
        if (gone) announce("Highlight removed", false, { label: "Undo", run: () => restore(gone) });
      } catch (err) {
        list.value = before;
        fail(err);
      }
    },
  };
}
