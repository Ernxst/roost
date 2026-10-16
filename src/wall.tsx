import { useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import type { ComponentChildren } from "preact";
import { Stamps } from "./card";
import { Library, type Bookmark, type Duplicates, type Media } from "./mygo";
import { count, fail, openPost, stats } from "./state";
import { Avatar, cached, mediaAlternative, shortDate } from "./tweet";

type Kind = "all" | "photo" | "video" | "animated_gif";
const kinds: [Kind, string][] = [
  ["all", "Everything"],
  ["photo", "Photos"],
  ["video", "Videos"],
  ["animated_gif", "GIFs"],
];
const emptyKind: Record<Kind, string> = { all: "photos, videos or GIFs", photo: "photos", video: "videos", animated_gif: "GIFs" };

/** Loads a list once per view, and again when a sync changes the library. */
export function useLoad<T>(load: () => Promise<T[]>) {
  const list = useSignal<T[] | null>(null);
  const total = stats.value?.total;
  useEffect(() => {
    load().then((v) => (list.value = v), (err) => fail(err));
  }, [total]);
  return list.value;
}

/** Every picture, video and GIF in the library, as a contact sheet. */
export function WallView() {
  const marks = useLoad(Library.wall);
  const kind = useSignal<Kind>("all");
  const tiles = (marks ?? []).flatMap((b) =>
    b.tweet.media.filter((m) => kind.value === "all" || m.kind === kind.value).map((m) => ({ b, m })),
  );
  return (
    <section class="wall-view" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">Media wall</h1>
        <span class="feed-count">{marks ? count(tiles.length, "piece") : "Loading…"}</span>
        <div class="wall-kinds" role="group" aria-label="Show">
          {kinds.map(([k, label]) => (
            <button key={k} class="chip" aria-pressed={kind.value === k} onClick={() => (kind.value = k)}>
              {label}
            </button>
          ))}
        </div>
      </div>
      {marks && tiles.length === 0 ? (
        <p class="empty">No {emptyKind[kind.value]} among your bookmarks yet.</p>
      ) : (
        <ul class="wall">
          {tiles.map(({ b, m }) => (
            <li key={`${b.tweet.id} ${m.thumb}`}>
              <Tile b={b} m={m} />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Tile({ b, m }: { b: Bookmark; m: Media }) {
  const t = b.tweet;
  const ratio = m.width && m.height ? Math.min(Math.max(m.width / m.height, 0.5), 2.2) : 1;
  return (
    <button class={`tile${b.removedAt ? " gone" : ""}`} style={{ aspectRatio: String(ratio) }} onClick={() => openPost(t.id)}>
      <img src={cached(`${m.thumb}?name=small`)} alt={mediaAlternative(m)} loading="lazy" />
      {m.kind !== "photo" && (
        <span class="badge" aria-hidden="true">
          {m.kind === "video" ? "▶" : "GIF"}
        </span>
      )}
      <span class="tile-caption">
        <span class="sr-only">{m.kind === "photo" ? "Photo" : m.kind === "video" ? "Video" : "GIF"} from </span>@{t.authorHandle}
      </span>
      {b.removedAt && (
        <span class="tile-gone">
          <span class="sr-only">, </span>missing from bookmarks
        </span>
      )}
    </button>
  );
}

const reasons: Record<string, (key: string) => ComponentChildren> = {
  post: () => "The same post",
  link: (key) => (
    <>
      The same page · <span class="dupe-url">{key.split("?")[0]}</span>
    </>
  ),
  image: () => "The same picture",
};

/** Bookmarks that point at the same post, page or picture. */
export function DuplicatesView() {
  const groups = useLoad<Duplicates>(Library.duplicates);
  return (
    <section class="dupes-view" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">Duplicates</h1>
        <span class="feed-count">{groups ? count(groups.length, "set") : "Loading…"}</span>
      </div>
      {groups && groups.length === 0 ? (
        <p class="empty">No two bookmarks point at the same thing.</p>
      ) : (
        <ol class="dupes">
          {groups?.map((g) => (
            <li key={`${g.reason} ${g.key}`} class="dupe">
              <h2 class="dupe-reason">{reasons[g.reason]?.(g.key) ?? g.key}</h2>
              <ul>
                {g.bookmarks.map((b) => (
                  <li key={b.tweet.id}>
                    <button class="dupe-row" onClick={() => openPost(b.tweet.id)}>
                      <Avatar src={b.tweet.authorAvatar} name={b.tweet.authorName} size={22} />
                      <span class="dupe-text">
                        <span class="name" dir="auto">{b.tweet.authorName}</span> {b.tweet.text.replace(/https?:\/\/\S+/g, "").trim() || "(no text)"}
                      </span>
                      <time dateTime={b.tweet.createdAt}>{shortDate(b.tweet.createdAt)}</time>
                    </button>
                    <Stamps b={b} />
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
