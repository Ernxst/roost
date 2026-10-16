import type { ComponentChildren, JSX } from "preact";
import { useState } from "preact/hooks";
import { Library, type Media, type Tweet } from "./mygo";

const entities: Record<string, string> = { "&amp;": "&", "&lt;": "<", "&gt;": ">", "&quot;": '"', "&#39;": "'" };
const decode = (s: string) => s.replace(/&(amp|lt|gt|quot|#39);/g, (m) => entities[m] ?? m);

export function openExternal(e: MouseEvent, url: string) {
  e.preventDefault();
  e.stopPropagation();
  Library.open(url);
}

function Ext({ href, children }: { href: string; children: JSX.Element | string }) {
  return (
    <a href={href} class="ext" onClick={(e) => openExternal(e, href)}>
      {children}
    </a>
  );
}

const languages = new Intl.DisplayNames(undefined, { type: "language" });
const languageName = (code: string) => {
  try {
    return languages.of(code) ?? code;
  } catch {
    return code;
  }
};

/**
 * Post text with t.co links expanded and mentions, hashtags and URLs linked.
 * A post in another language shows X's translation, with the original a
 * click away. `mark` wraps the plain text between links, such as to show
 * highlights.
 */
export function Text({ tweet, mark = (s) => s }: { tweet: Tweet; mark?: (text: string) => ComponentChildren }) {
  const [original, setOriginal] = useState(false);
  const tr = tweet.translation;
  const translated = tr && !original;
  return (
    <>
      <Linked tweet={tweet} mark={mark} text={translated ? tr.text : tweet.text} lang={translated ? tr.to : tr?.from} />
      {tr && (
        <p class="translated">
          {translated ? `Translated from ${languageName(tr.from)}` : `Written in ${languageName(tr.from)}`}
          {" · "}
          <button type="button" onClick={() => setOriginal(!original)}>
            {original ? "Show translation" : "Show original"}
          </button>
        </p>
      )}
    </>
  );
}

function Linked({
  tweet,
  text: raw,
  lang,
  mark,
}: {
  tweet: Tweet;
  text: string;
  lang?: string;
  mark: (text: string) => ComponentChildren;
}) {
  const links = new Map(tweet.links.map((l) => [l.short, l]));
  const out: ComponentChildren[] = [];
  const pattern = /(https?:\/\/\S+)|(@\w{1,15})|(#[\p{L}\p{N}_]+)/gu;
  const text = decode(raw);
  let last = 0;
  for (const m of text.matchAll(pattern)) {
    out.push(mark(text.slice(last, m.index)));
    const [token, url, mention, tag] = m;
    if (url) {
      let clean = url.replace(/[.,"'”’]+$/u, "");
      while (clean.endsWith(")") && (clean.match(/\)/g)?.length ?? 0) > (clean.match(/\(/g)?.length ?? 0)) clean = clean.slice(0, -1);
      const l = links.get(clean);
      out.push(<Ext href={l?.expanded ?? clean}>{l?.display ?? clean}</Ext>);
      out.push(url.slice(clean.length));
    } else if (mention) {
      out.push(<Ext href={`https://x.com/${mention.slice(1)}`}>{mention}</Ext>);
    } else if (tag) {
      out.push(<Ext href={`https://x.com/hashtag/${encodeURIComponent(tag.slice(1))}`}>{tag}</Ext>);
    }
    last = (m.index ?? 0) + token.length;
  }
  out.push(mark(text.slice(last)));
  return (
    <p class="text" dir="auto" lang={lang}>
      {out}
    </p>
  );
}

const dayFormat = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" });
const longDayFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "long" });
const longFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "long", timeStyle: "short" });

/** A date written out, such as 14 October 2026. */
export const longDay = (iso: string) => longDayFormat.format(new Date(iso));
/** A date and time written out, such as 14 October 2026 at 09:30. */
export const longDate = (iso: string) => longFormat.format(new Date(iso));

// A minute of clock skew still reads as recent; anything later shows its date.
const SKEW = 6e4;

export function shortDate(iso: string) {
  const d = new Date(iso);
  const ago = Date.now() - d.getTime();
  if (ago < -SKEW) return dayFormat.format(d);
  const hours = ago / 36e5;
  if (hours < 1) return `${Math.max(1, Math.round(hours * 60))}m`;
  if (hours < 24) return `${Math.round(hours)}h`;
  return dayFormat.format(d);
}

export function relative(iso: string) {
  const ago = Date.now() - new Date(iso).getTime();
  if (ago < -SKEW) return dayFormat.format(new Date(iso));
  const mins = Math.round(ago / 6e4);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours} h ago`;
  return dayFormat.format(new Date(iso));
}

/** Loads X media through roost's on-disk copy, fetching it the first time. */
export const cached = (url: string) => `roost-media://localhost/?u=${encodeURIComponent(url)}`;

export function Avatar({ src, name = "", size = 28 }: { src: string; name?: string; size?: number }) {
  // Remembers which address failed, so a new author's avatar gets its own try.
  const [failed, setFailed] = useState("");
  if (!src || failed === src) {
    return (
      <span class="avatar blank" style={{ width: `${size}px`, height: `${size}px`, fontSize: `${Math.round(size * 0.46)}px` }} aria-hidden="true">
        {initial(name)}
      </span>
    );
  }
  return <img class="avatar" src={cached(src)} width={size} height={size} alt="" loading="lazy" onError={() => setFailed(src)} />;
}

const graphemes = new Intl.Segmenter(undefined, { granularity: "grapheme" });
function initial(name: string) {
  for (const { segment } of graphemes.segment(name.trim())) {
    if (/\p{L}/u.test(segment)) return segment.toLocaleUpperCase();
  }
  return "?";
}

/**
 * The aspect ratio a lone photo or video is framed at. It is the media's own,
 * clamped between portrait 3:4 and 2:1 so panoramas and tall screenshots stay
 * readable. Cards crop to the frame; the detail sheet letterboxes inside it
 * to show the whole image.
 */
export function previewRatio(m: Media): number | undefined {
  if (!m.width || !m.height) return undefined;
  return Math.min(Math.max(m.width / m.height, 0.75), 2);
}

export function mediaAlternative(m: Media): string {
  if (m.description?.trim()) return m.description.trim();
  const kind = m.kind === "photo" ? "Photo" : m.kind === "video" ? "Video" : "GIF";
  return `${kind} attachment; no description was provided`;
}

/** Media laid out as X does: one item at its own shape, more in a grid. */
export function MediaGrid({ media, playable = false }: { media: Media[]; playable?: boolean }) {
  if (media.length === 0) return null;
  const single = media.length === 1;
  return (
    <div class={`media n${Math.min(media.length, 4)}`}>
      {media.slice(0, 4).map((m) => {
        const ratio = single ? previewRatio(m) : undefined;
        const style = ratio ? { aspectRatio: String(ratio) } : undefined;
        if (playable && m.video) {
          return (
            <video
              key={m.thumb}
              src={cached(m.video)}
              poster={cached(m.thumb)}
              controls
              playsInline
              preload="metadata"
              loop={m.kind === "animated_gif"}
              aria-label={mediaAlternative(m)}
              style={style}
            />
          );
        }
        return (
          <figure key={m.thumb} class="frame" style={style}>
            <img src={cached(`${m.thumb}?name=small`)} alt={mediaAlternative(m)} loading="lazy" />
            {m.kind !== "photo" && (
              <span class="badge">
                {m.kind === "video" ? (
                  <>
                    <span aria-hidden="true">▶</span>
                    <span class="sr-only">Video</span>
                  </>
                ) : (
                  "GIF"
                )}
              </span>
            )}
          </figure>
        );
      })}
    </div>
  );
}

export function Quoted({ tweet }: { tweet: Tweet }) {
  return (
    <blockquote class="quoted">
      <div class="byline">
        <Avatar src={tweet.authorAvatar} name={tweet.authorName} size={18} />
        <span class="name" dir="auto">{tweet.authorName}</span>
        <span class="handle">@{tweet.authorHandle}</span>
      </div>
      <Text tweet={tweet} />
      <MediaGrid media={tweet.media} />
    </blockquote>
  );
}
