import { signal, useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import { ink } from "./card";
import { Library, type Author, type AuthorSummary } from "./mygo";
import { collections, count, fail, search, stats, view } from "./state";
import { Avatar, openExternal } from "./tweet";
import { useLoad } from "./wall";

/** The author whose posts the feed shows, once loaded. */
export const author = signal<AuthorSummary | null>(null);

/** Shows an author's posts. */
export function showAuthor(handle: string) {
  search.value = "";
  view.value = { kind: "author", handle };
}

/** Everyone the user has kept a post by, most kept first. */
export function AuthorsView() {
  const all = useLoad<Author>(() => Library.authors(-1));
  const filter = useSignal("");
  const q = filter.value.trim().toLowerCase().replace(/^@/, "");
  const shown = (all ?? []).filter((a) => !q || a.handle.toLowerCase().includes(q) || a.name.toLowerCase().includes(q));
  return (
    <section class="authors-view" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">Authors</h1>
        <span class="feed-count">{all ? count(shown.length, "author") : "…"}</span>
        <label class="authors-filter">
          <span class="sr-only">Find an author</span>
          <input
            type="search"
            value={filter.value}
            onInput={(e) => (filter.value = e.currentTarget.value)}
            placeholder="Find an author…"
            autoComplete="off"
            spellcheck={false}
          />
        </label>
      </div>
      {all && shown.length === 0 ? (
        <p class="empty">{q ? "No author by that name." : "Nothing here yet."}</p>
      ) : (
        <ul class="authors-grid">
          {shown.map((a) => (
            <li key={a.handle}>
              <button class="author-row" onClick={() => showAuthor(a.handle)}>
                <Avatar src={a.avatar} name={a.name} size={32} />
                <span class="who">
                  <span class="name">{a.name || a.handle}</span>
                  <span class="handle">@{a.handle}</span>
                </span>
                <span class="count">{a.count.toLocaleString()}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

/**
 * What the library holds by the author in view: a link to their profile,
 * how many of their bookmarks are missing, and the collections their posts
 * are filed in, each of which narrows the feed to it.
 */
export function AuthorHead({ handle }: { handle: string }) {
  const total = stats.value?.total;
  useEffect(() => {
    author.value = null;
    Library.authorSummary(handle).then(
      (a) => (author.value = a),
      fail,
    );
  }, [handle, total]);
  const a = author.value;
  if (!a || a.handle.toLowerCase() !== handle.toLowerCase()) return null;
  const byId = new Map(collections.value.map((c) => [c.id, c]));
  const profile = `https://x.com/${a.handle}`;
  const pick = (q: string) => (search.value = search.value === q ? "" : q);
  return (
    <div class="author-head">
      <Avatar src={a.avatar} name={a.name || a.handle} size={44} />
      <div class="author-facts">
        <a class="ext" href={profile} onClick={(e) => openExternal(e, profile)}>
          @{a.handle} on X ↗
        </a>
        {(a.gone > 0 || a.collections.length > 0) && (
          <div class="author-chips" role="group" aria-label="Narrow to">
            {a.collections.map(({ id, count }) => {
              const c = byId.get(id);
              if (!c) return null;
              const q = `in:"${c.name.replaceAll('"', '\\"')}"`;
              return (
                <button key={id} class="chip filed" title={c.name} style={{ "--hue": ink(c.hue) }} aria-pressed={search.value === q} onClick={() => pick(q)}>
                  {c.name} · {count.toLocaleString()}
                </button>
              );
            })}
            {a.gone > 0 && (
              <button class="chip" aria-pressed={search.value === "is:gone"} onClick={() => pick("is:gone")}>
                Missing from bookmarks · {a.gone.toLocaleString()}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
