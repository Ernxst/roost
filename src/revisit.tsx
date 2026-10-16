import type { Bookmark } from "./mygo";
import { Card } from "./card";
import { loadRevisit, revisit } from "./state";

/** Revisit brings back posts worth a second look instead of letting them sink. */
export function RevisitView() {
  const r = revisit.value;
  return (
    <section class="revisit" aria-labelledby="feed-title">
      <div class="feed-head">
        <h1 id="feed-title">Revisit</h1>
        <span class="feed-count">posts worth a second look</span>
      </div>
      {!r ? (
        <p class="empty">…</p>
      ) : (
        <>
          <Shelf title="On this day" note="posted on this date in earlier years" items={r.onThisDay} />
          <Shelf title="Never opened" note="kept, then never looked at in roost" items={r.unopened} />
          <Shelf
            title="A random handful"
            items={r.random}
            action={
              <button class="press quiet" onClick={loadRevisit}>
                Shuffle
              </button>
            }
          />
        </>
      )}
    </section>
  );
}

function Shelf({ title, note, items, action }: { title: string; note?: string; items: Bookmark[]; action?: preact.ComponentChildren }) {
  if (items.length === 0 && !action) return null;
  return (
    <section class="shelf" aria-label={title}>
      <div class="shelf-head">
        <h2>{title}</h2>
        {note && <span class="feed-count">{note}</span>}
        {action}
      </div>
      <div class="shelf-row">
        {items.map((b) => (
          <Card key={b.tweet.id} b={b} />
        ))}
      </div>
    </section>
  );
}
