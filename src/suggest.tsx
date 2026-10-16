import { useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import { ink } from "./card";
import { Library, type Bookmark } from "./mygo";
import { collections } from "./state";

/** The collections a post is likely to belong in, best first. */
export function useSuggestions(b: Bookmark | undefined) {
  const ids = useSignal<number[]>([]);
  const filed = b?.collections.join(",");
  useEffect(() => {
    ids.value = [];
    if (b) Library.suggest(b.tweet.id).then((s) => (ids.value = s.map((x) => x.collectionId)), () => {});
  }, [b?.tweet.id, filed, collections.value.length]);
  return ids;
}

/** One-click chips for the suggested collections. */
export function SuggestedChips({ ids, onFile }: { ids: number[]; onFile: (id: number) => void }) {
  const byId = new Map(collections.value.map((c) => [c.id, c]));
  const list = ids.map((id) => byId.get(id)).filter((c) => !!c);
  if (list.length === 0) return null;
  return (
    <p class="suggested">
      <span class="suggested-label">Looks like</span>
      {list.map((c) => (
        <button key={c.id} class="suggestion" style={{ "--hue": ink(c.hue) }} onClick={() => onFile(c.id)}>
          + {c.name}
        </button>
      ))}
    </p>
  );
}
