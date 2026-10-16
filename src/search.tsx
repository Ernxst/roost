import { useComputed, useSignal } from "@preact/signals";
import { useRef } from "preact/hooks";
import { accept, complete, type Suggestion } from "./autocomplete";
import { Library, type Author } from "./mygo";
import { authors, collections, count, search, stats } from "./state";

const HINTS: [string, string][] = [
  ["from:", "posts by an author"],
  ["has:video", "also photo, gif, link, quote, note, thread"],
  ["in:", "in a collection; in:none for unsorted"],
  ["-", "leave out a word or filter"],
  ["before:2025-01-01", "or after:, by post date"],
  ["is:gone", "missing from bookmarks"],
  ["domain:github.com", "posts linking to a site"],
  ["lang:en", "written in a language"],
  ["likes:1000", "at least that many likes"],
];

/**
 * The search field, a combobox: an empty field lists the operators, and a
 * word being typed lists completions for it. Arrow keys move through them,
 * Tab or Enter takes one, Escape closes the list and then clears the field.
 */
export function SearchBox() {
  const input = useRef<HTMLInputElement>(null);
  const focused = useSignal(false);
  const caret = useSignal(0);
  const active = useSignal(-1);
  const dismissed = useSignal(false);
  // The rail shows only the most-kept authors; completion wants all of them.
  const everyone = useSignal<Author[] | null>(null);

  const completions = useComputed(() =>
    complete(search.value, caret.value, { authors: everyone.value ?? authors.value, collections: collections.value }),
  );
  const listed = useComputed(() => (focused.value && !dismissed.value ? completions.value.items : []));

  function focus() {
    focused.value = true;
    if (!everyone.value) Library.authors(-1).then((a) => (everyone.value = a));
  }

  function sync(el: HTMLInputElement) {
    caret.value = el.selectionStart ?? el.value.length;
  }

  function take(s: Suggestion) {
    const { value, caret: at } = accept(search.value, completions.value, s);
    search.value = value;
    caret.value = at;
    active.value = -1;
    const el = input.current;
    if (el) {
      el.value = value;
      el.setSelectionRange(at, at);
      el.focus();
    }
  }

  function onKeyDown(e: KeyboardEvent) {
    const items = listed.value;
    if (items.length && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
      e.preventDefault();
      // Cycles through the items and back to none, which leaves the typing alone.
      const n = items.length;
      active.value = ((active.value + 1 + (e.key === "ArrowDown" ? 1 : -1) + n + 1) % (n + 1)) - 1;
    } else if (items.length && (e.key === "Tab" || e.key === "Enter") && !e.shiftKey) {
      const pick = items[active.value] ?? (e.key === "Tab" ? items[0] : undefined);
      if (pick) {
        e.preventDefault();
        take(pick);
      }
    } else if (e.key === "Escape") {
      if (items.length) {
        dismissed.value = true;
        active.value = -1;
      } else {
        search.value = "";
        input.current?.blur();
      }
    }
  }

  const items = listed.value;
  const open = items.length > 0;
  return (
    <div
      class="search"
      onFocusIn={focus}
      onFocusOut={() => {
        focused.value = false;
        active.value = -1;
      }}
    >
      <label for="search" class="sr-only">
        Search bookmarks
      </label>
      <span class="glyph" aria-hidden="true">
        ⌕
      </span>
      <input
        id="search"
        ref={input}
        type="text"
        enterKeyHint="search"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={open}
        aria-controls="search-suggestions"
        aria-activedescendant={open && active.value >= 0 ? `suggestion-${active.value}` : undefined}
        value={search.value}
        onInput={(e) => {
          search.value = e.currentTarget.value;
          sync(e.currentTarget);
          active.value = -1;
          dismissed.value = false;
        }}
        onKeyUp={(e) => sync(e.currentTarget)}
        onClick={(e) => sync(e.currentTarget)}
        onKeyDown={onKeyDown}
        placeholder={stats.value?.total ? `Search ${count(stats.value.total, "kept post")}` : "Search kept posts"}
        autoComplete="off"
        spellcheck={false}
        aria-describedby="search-hints"
      />
      <kbd aria-hidden="true">/</kbd>
      <SearchHints
        open={focused.value && !search.value}
        onPick={(op) => {
          search.value = op;
          caret.value = op.length;
          input.current?.focus();
        }}
      />
      <div class="hints" hidden={!open}>
        <p class="hints-title" aria-hidden="true">
          Tab to complete
        </p>
        <ul id="search-suggestions" role="listbox" aria-label="Completions">
          {items.map((s, i) => (
            <li
              key={s.label}
              id={`suggestion-${i}`}
              role="option"
              aria-selected={i === active.value}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => take(s)}
            >
              <code>{s.label}</code>
              <span>{s.detail}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function SearchHints({ open, onPick }: { open: boolean; onPick: (op: string) => void }) {
  return (
    <div id="search-hints" class="hints" hidden={!open}>
      <p class="hints-title">Narrow it down</p>
      <ul>
        {HINTS.map(([op, what]) => (
          <li key={op}>
            <button
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => onPick(op)}
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
