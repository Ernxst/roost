import type { JSX } from "preact";
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

/** Post text with t.co links expanded and mentions, hashtags and URLs linked. */
export function Text({ tweet }: { tweet: Tweet }) {
  const links = new Map(tweet.links.map((l) => [l.short, l]));
  const out: (JSX.Element | string)[] = [];
  const pattern = /(https?:\/\/\S+)|(@\w{1,15})|(#[\p{L}\p{N}_]+)/gu;
  const text = decode(tweet.text);
  let last = 0;
  for (const m of text.matchAll(pattern)) {
    out.push(text.slice(last, m.index));
    const [token, url, mention, tag] = m;
    if (url) {
      const l = links.get(url);
      out.push(<Ext href={l?.expanded ?? url}>{l?.display ?? url}</Ext>);
    } else if (mention) {
      out.push(<Ext href={`https://x.com/${mention.slice(1)}`}>{mention}</Ext>);
    } else if (tag) {
      out.push(<Ext href={`https://x.com/hashtag/${encodeURIComponent(tag.slice(1))}`}>{tag}</Ext>);
    }
    last = (m.index ?? 0) + token.length;
  }
  out.push(text.slice(last));
  return <p class="text">{out}</p>;
}

const dayFormat = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" });

export function shortDate(iso: string) {
  const d = new Date(iso);
  const hours = (Date.now() - d.getTime()) / 36e5;
  if (hours < 1) return `${Math.max(1, Math.round(hours * 60))}m`;
  if (hours < 24) return `${Math.round(hours)}h`;
  return dayFormat.format(d);
}

export function relative(iso: string) {
  const mins = Math.round((Date.now() - new Date(iso).getTime()) / 6e4);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours} h ago`;
  return dayFormat.format(new Date(iso));
}

export function Avatar({ src, size = 28 }: { src: string; size?: number }) {
  if (!src) return <span class="avatar blank" style={{ width: size, height: size }} aria-hidden="true" />;
  return <img class="avatar" src={src} width={size} height={size} alt="" loading="lazy" />;
}

/** Media laid out as X does: one item at its own shape, more in a grid. */
export function MediaGrid({ media, playable = false }: { media: Media[]; playable?: boolean }) {
  if (media.length === 0) return null;
  const single = media.length === 1;
  return (
    <div class={`media n${Math.min(media.length, 4)}`}>
      {media.slice(0, 4).map((m) => {
        const ratio = single && m.width && m.height ? Math.max(m.width / m.height, 0.75) : undefined;
        const style = ratio ? { aspectRatio: String(ratio) } : undefined;
        if (playable && m.video) {
          return (
            <video
              key={m.thumb}
              src={m.video}
              poster={m.thumb}
              controls
              playsInline
              loop={m.kind === "animated_gif"}
              style={style}
            />
          );
        }
        return (
          <figure key={m.thumb} class="frame" style={style}>
            <img src={`${m.thumb}?name=small`} alt="" loading="lazy" />
            {m.kind !== "photo" && (
              <span class="badge" aria-label={m.kind === "video" ? "Video" : "GIF"}>
                {m.kind === "video" ? "▶" : "GIF"}
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
        <Avatar src={tweet.authorAvatar} size={18} />
        <span class="name">{tweet.authorName}</span>
        <span class="handle">@{tweet.authorHandle}</span>
      </div>
      <Text tweet={tweet} />
      <MediaGrid media={tweet.media} />
    </blockquote>
  );
}
