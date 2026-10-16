import { useComputed } from "@preact/signals";
import { Component } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import type { Bookmark, Collection } from "./mygo";
import { collections, cursor, opened, selected, toggleSelected } from "./state";
import { Avatar, MediaGrid, Quoted, shortDate, Text } from "./tweet";

/** The ink colour of a collection's hue, dark enough on a card in either scheme to read as text. */
export const ink = (hue: number) => `light-dark(oklch(0.48 0.14 ${hue}), oklch(0.68 0.14 ${hue}))`;

export function Stamps({ b, compact = false }: { b: Bookmark; compact?: boolean }) {
  const byId = new Map(collections.value.map((c) => [c.id, c]));
  const cs = b.collections.map((id) => byId.get(id)).filter((c): c is Collection => !!c);
  if (cs.length === 0) return null;
  return (
    <ul class={`stamps${compact ? " compact" : ""}`} aria-label="Collections" title={cs.map((c) => c.name).join(", ")}>
      {(compact ? cs.slice(0, 2) : cs).map((c, i) => (
        <li key={c.id} class="stamp" style={{ "--hue": ink(c.hue), "--tilt": `${((c.id * 7 + i * 3) % 5) - 2}deg` }}>
          {c.name}
        </li>
      ))}
      {compact && cs.length > 2 && <li class="stamp stamp-more">+{cs.length - 2}</li>}
    </ul>
  );
}

/**
 * A feed card that re-renders only when its bookmark changes. With a thousand
 * cards on the desk, the feed re-rendering for a page load or a keypress
 * leaves every unchanged card alone.
 */
export class Card extends Component<{ b: Bookmark }> {
  shouldComponentUpdate(next: { b: Bookmark }) {
    return next.b !== this.props.b;
  }
  render() {
    return <CardBody b={this.props.b} />;
  }
}

function CardBody({ b }: { b: Bookmark }) {
  const t = b.tweet;
  const ref = useRef<HTMLElement>(null);
  const [clipped, setClipped] = useState(false);
  // Watches only whether this card holds the cursor, so moving it re-renders two cards.
  const active = useComputed(() => cursor.value === t.id).value;
  // The cursor moves by key, often with the key held down, so the feed jumps rather than glides to it.
  const picked = useComputed(() => selected.value.has(t.id)).value;
  const picking = useComputed(() => selected.value.size > 0).value;
  useEffect(() => {
    if (active) ref.current?.scrollIntoView({ block: "nearest" });
  }, [active]);
  useEffect(() => {
    const text = ref.current?.querySelector<HTMLElement>(":scope > .text");
    if (!text) {
      setClipped(false);
      return;
    }
    const measure = () => setClipped(text.scrollHeight > text.clientHeight + 1);
    const observer = new ResizeObserver(measure);
    observer.observe(text);
    window.addEventListener("resize", measure);
    measure();
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", measure);
    };
  }, [t.id, t.text]);
  return (
    <article
      ref={ref}
      class={`card${active ? " cursor" : ""}${b.removedAt ? " gone" : ""}${picked ? " picked" : ""}${picking ? " picking" : ""}`}
      aria-labelledby={`by-${t.id}`}
    >
      <header class="byline">
        <Avatar src={t.authorAvatar} name={t.authorName || t.authorHandle} />
        <span class="who" id={`by-${t.id}`}>
          <span class="name" dir="auto">{t.authorName}</span>
          <span class="handle" title={`@${t.authorHandle}`}>@{t.authorHandle}</span>
        </span>
        <time dateTime={t.createdAt}>{shortDate(t.createdAt)}</time>
        <label class="pick">
          <input type="checkbox" checked={picked} onChange={() => toggleSelected(t.id)} />
          <span class="sr-only">Select post by {t.authorName || `@${t.authorHandle}`}</span>
        </label>
      </header>
      {b.threadLength > 1 && <span class="thread-tag">thread · {b.threadLength} posts</span>}
      <button class="open" onClick={() => ((opened.value = t.id), (cursor.value = t.id))}>
        <span class="sr-only">Open post by {t.authorName || `@${t.authorHandle}`}</span>
      </button>
      {t.text && <Text tweet={t} />}
      {clipped && <button class="show-more" onClick={() => ((opened.value = t.id), (cursor.value = t.id))}>Show more</button>}
      <MediaGrid media={t.media} />
      {t.quoted && <Quoted tweet={t.quoted} />}
      {b.note && <p class="note-preview" dir="auto">{b.note}</p>}
      <Stamps b={b} />
      {b.removedAt && (
        <span class="gone-stamp">missing from bookmarks</span>
      )}
    </article>
  );
}
