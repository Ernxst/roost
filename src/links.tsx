import { useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import { Library, type Bookmark, type Page, type SavedLink } from "./mygo";
import { fail } from "./state";
import { Highlighter, marked } from "./highlights";
import type { Highlight } from "./mygo";
import { openExternal } from "./tweet";

/** The pages a post links to, with the saved copy of each where there is one. */
export function Links({ b, onRead }: { b: Bookmark; onRead: (url: string) => void }) {
  const list = useSignal<SavedLink[]>([]);
  useEffect(() => {
    list.value = [];
    if (b.tweet.links.length)
      Library.links(b.tweet.id).then(
        (l) => (list.value = l.filter((x) => !/^https?:\/\/(www\.)?(x|twitter)\.com\//.test(x.expanded))),
      );
  }, [b.tweet.id]);
  if (list.value.length === 0) return null;
  return (
    <section class="links" aria-label="Links">
      <h3 class="legend">Linked</h3>
      <ul>
        {list.value.map((l) => (
          <li key={l.expanded} class="link-card">
            <span class="site">{l.page?.site || l.display.split("/")[0]}</span>
            <a class="title" href={l.expanded} onClick={(e) => openExternal(e, l.expanded)}>
              {l.page?.title || l.display}
            </a>
            {l.page?.excerpt && <p class="excerpt">{l.page.excerpt}</p>}
            {l.hasText ? (
              <button class="add" onClick={() => onRead(l.expanded)}>
                Read the saved copy
              </button>
            ) : (
              <div class="link-status">
                {l.page ? <>{linkError(l.page.error ?? "")}{l.page.error && <details><summary>Details</summary><code>{l.page.error}</code></details>}</> : "Not fetched yet"}
              </div>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}

function linkError(error: string) {
  if (!error) return "Not saved: page text could not be read.";
  if (/timeout|deadline/i.test(error)) return "Not saved: the site took too long to respond.";
  if (/tls|certificate|handshake/i.test(error)) return "Not saved: the site’s secure connection failed.";
  if (/lookup|dial tcp|connection refused|network/i.test(error)) return "Not saved: the site could not be reached.";
  if (/sign.?in|login|unauthori[sz]ed|forbidden/i.test(error)) return "Not saved: the page may require signing in.";
  return "Not saved: the page could not be read.";
}

/** A saved page as plain text, set for reading. */
export function Reader({
  url,
  tweetId,
  quotes,
  onAdd,
  onClose,
}: {
  url: string;
  tweetId: string;
  quotes: string[];
  onAdd: (h: Highlight) => void;
  onClose: () => void;
}) {
  const page = useSignal<Page | null>(null);
  useEffect(() => {
    Library.page(url)
      .then((p) => (page.value = p))
      .catch((err) => fail(err));
  }, [url]);
  const p = page.value;
  return (
    <div class="reader">
      <button class="add" onClick={onClose}>
        ← Back to the post
      </button>
      {p && (
        <Highlighter tweetId={tweetId} source={url} onAdd={onAdd}>
          <article>
            <p class="site">{p.site}</p>
            <h2>{p.title}</h2>
            {p.byline && <p class="byline-text">{p.byline}</p>}
            {p.text?.split(/\n{2,}/).map((para, i) => (
              <p key={i}>{marked(para, quotes)}</p>
            ))}
            <a class="press link quiet" href={p.finalUrl} onClick={(e) => openExternal(e, p.finalUrl)}>
              Open the original ↗
            </a>
          </article>
        </Highlighter>
      )}
    </div>
  );
}
