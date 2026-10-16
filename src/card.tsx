import { useEffect, useRef } from "preact/hooks";
import type { Bookmark, Collection } from "./mygo";
import { collections, cursor, opened } from "./state";
import { Avatar, MediaGrid, Quoted, shortDate, Text } from "./tweet";

/** The ink colour of a collection's hue. */
export const ink = (hue: number) => `oklch(0.56 0.14 ${hue})`;

export function Stamps({ b }: { b: Bookmark }) {
  const byId = new Map(collections.value.map((c) => [c.id, c]));
  const cs = b.collections.map((id) => byId.get(id)).filter((c): c is Collection => !!c);
  if (cs.length === 0) return null;
  return (
    <ul class="stamps" aria-label="Collections">
      {cs.map((c, i) => (
        <li key={c.id} class="stamp" style={{ "--hue": ink(c.hue), "--tilt": `${((c.id * 7 + i * 3) % 5) - 2}deg` }}>
          {c.name}
        </li>
      ))}
    </ul>
  );
}

export function Card({ b }: { b: Bookmark }) {
  const t = b.tweet;
  const ref = useRef<HTMLElement>(null);
  const active = cursor.value === t.id;
  useEffect(() => {
    if (active) ref.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [active]);
  return (
    <article
      ref={ref}
      class={`card${active ? " cursor" : ""}${b.removedAt ? " gone" : ""}`}
      aria-labelledby={`by-${t.id}`}
    >
      <header class="byline">
        <Avatar src={t.authorAvatar} />
        <span class="who" id={`by-${t.id}`}>
          <span class="name">{t.authorName}</span>
          <span class="handle">@{t.authorHandle}</span>
        </span>
        <time dateTime={t.createdAt}>{shortDate(t.createdAt)}</time>
      </header>
      <button class="open" onClick={() => ((opened.value = t.id), (cursor.value = t.id))}>
        <span class="sr-only">Open post by {t.authorName}</span>
      </button>
      {t.text && <Text tweet={t} />}
      <MediaGrid media={t.media} />
      {t.quoted && <Quoted tweet={t.quoted} />}
      {b.note && <p class="note-preview">{b.note}</p>}
      <Stamps b={b} />
      {b.removedAt && (
        <span class="gone-stamp" aria-label="Gone from X">
          gone from X
        </span>
      )}
    </article>
  );
}
