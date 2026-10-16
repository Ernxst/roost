import { useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import { Library, type Bookmark, type Tweet } from "./mygo";
import { announce, fail, reload } from "./state";
import { MediaGrid, Quoted, shortDate, Text } from "./tweet";

/** The rest of a post's thread, read in order, with the kept post marked. */
export function Thread({ b }: { b: Bookmark }) {
  const posts = useSignal<Tweet[] | null>(null);
  const busy = useSignal(false);

  useEffect(() => {
    posts.value = null;
    if (b.threadLength > 1) Library.thread(b.tweet.id).then((p) => (posts.value = p));
  }, [b.tweet.id, b.threadLength]);

  async function look() {
    busy.value = true;
    try {
      const p = await Library.loadThread(b.tweet.id, b.tweet.url);
      posts.value = p;
      announce(p.length > 1 ? `Found a thread of ${p.length} posts` : "This post stands alone");
      if (p.length > 1) reload();
    } catch (err) {
      fail(err);
    } finally {
      busy.value = false;
    }
  }

  const list = posts.value;
  if (!list || list.length < 2) {
    return (
      <p class="thread-look">
        {list && !busy.value && <span class="link-status">This post stands alone.</span>}
        <button class="add" onClick={look} disabled={busy.value}>
          {busy.value ? "Reading X…" : list ? "Look again" : "Look for a thread"}
        </button>
      </p>
    );
  }
  return (
    <section class="thread" aria-label={`Thread of ${list.length} posts`}>
      <h3 class="legend">Thread · {list.length} posts</h3>
      <ol>
        {list.map((t, i) => (
          <li key={t.id} class={t.id === b.tweet.id ? "kept" : undefined} aria-current={t.id === b.tweet.id ? "true" : undefined}>
            <span class="thread-n" aria-hidden="true">
              {i + 1}
            </span>
            <div>
              {t.text && <Text tweet={t} />}
              <MediaGrid media={t.media} />
              {t.quoted && <Quoted tweet={t.quoted} />}
              <p class="thread-when">
                <time dateTime={t.createdAt}>{shortDate(t.createdAt)}</time>
                {t.id === b.tweet.id && <span class="thread-kept"> · the one you kept</span>}
              </p>
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}
