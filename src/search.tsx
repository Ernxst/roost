import { useSignal } from "@preact/signals";
import { useRef } from "preact/hooks";
import { count, search, stats } from "./state";

const HINTS: [string, string][] = [
  ["from:", "posts by an author"],
  ["has:video", "also photo, gif, link, quote, note, thread"],
  ["in:", "in a collection; in:none for unsorted"],
  ["-", "leave out a word or filter"],
  ["before:2025-01-01", "or after:, by post date"],
  ["is:gone", "removed on X"],
];

/** The search field, and the operators it lists while empty. */
export function SearchBox() {
  const input = useRef<HTMLInputElement>(null);
  const focused = useSignal(false);
  return (
    <div class="search" onFocusIn={() => (focused.value = true)} onFocusOut={() => (focused.value = false)}>
      <label for="search" class="sr-only">
        Search bookmarks
      </label>
      <span class="glyph" aria-hidden="true">
        ⌕
      </span>
      <input
        id="search"
        ref={input}
        type="search"
        value={search.value}
        onInput={(e) => (search.value = e.currentTarget.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            search.value = "";
            input.current?.blur();
          }
        }}
        placeholder={stats.value?.total ? `Search ${count(stats.value.total, "kept post")}` : "Search kept posts"}
        autoComplete="off"
        spellcheck={false}
        aria-describedby="search-hints"
      />
      <kbd aria-hidden="true">/</kbd>
      <SearchHints open={focused.value && !search.value} input={input} />
    </div>
  );
}

function SearchHints({ open, input }: { open: boolean; input: { current: HTMLInputElement | null } }) {
  return (
    <div id="search-hints" class="hints" hidden={!open}>
      <p class="hints-title">Narrow it down</p>
      <ul>
        {HINTS.map(([op, what]) => (
          <li key={op}>
            <button
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                search.value = op;
                input.current?.focus();
              }}
            >
              <code>{op}</code>
              <span>{what}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
