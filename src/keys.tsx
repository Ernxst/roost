import { useEffect, useRef } from "preact/hooks";
import type { ComponentChildren } from "preact";
import { manual } from "./state";

type Key = [keys: ComponentChildren, does: string];

const FEED: Key[] = [
  [<kbd>/</kbd>, "Search"],
  [
    <>
      <kbd>j</kbd> <kbd>k</kbd>
    </>,
    "Next or previous post",
  ],
  [<kbd>Enter</kbd>, "Open the post"],
  [<kbd>o</kbd>, "Open it on X"],
  [<kbd>Esc</kbd>, "Close the post"],
];

const SORT: Key[] = [
  [<kbd>s</kbd>, "Skip"],
  [<kbd>x</kbd>, "Set aside"],
  [<kbd>u</kbd>, "Undo"],
  [<kbd>n</kbd>, "New collection"],
  [
    <>
      <kbd>Shift</kbd> <kbd>1</kbd>–<kbd>9</kbd>
    </>,
    "File without moving on",
  ],
];

const READING: Key[] = [[<kbd>h</kbd>, "Highlight the selected text"]];

const APP: Key[] = [
  [
    <>
      <kbd>⌘</kbd> <kbd>,</kbd>
    </>,
    "Settings",
  ],
  [<kbd>?</kbd>, "These keys"],
];

/** Whether a key was pressed inside an open dialog, which handles its own keys. */
export function inDialog(e: KeyboardEvent) {
  return !!(e.target as Element).closest?.("dialog[open]");
}

function List({ title, keys }: { title: string; keys: Key[] }) {
  return (
    <section class="keys-group">
      <h3 class="legend">{title}</h3>
      <dl>
        {keys.map(([k, does]) => (
          <div key={does}>
            <dt>{k}</dt>
            <dd>{does}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

/** The Keys button in the rail, and the sheet of shortcuts it opens; ? opens it too. */
export function Keys() {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      const el = e.target as HTMLElement;
      if (e.key !== "?" || e.metaKey || e.ctrlKey || inDialog(e) || el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName)) return;
      e.preventDefault();
      dialog.current?.showModal();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const bins = manual().slice(0, 9);
  return (
    <>
      <button class="add" onClick={() => dialog.current?.showModal()} aria-keyshortcuts="?">
        Keys
      </button>
      <dialog class="settings keys" ref={dialog} aria-labelledby="keys-title" onClick={(e) => e.target === dialog.current && dialog.current?.close()}>
        <div class="settings-body">
          <div class="sheet-head">
            <h2 id="keys-title" class="sheet-kicker">
              Keys
            </h2>
            <button class="close" onClick={() => dialog.current?.close()} aria-label="Close">
              ✕
            </button>
          </div>
          <List title="Feed and post" keys={FEED} />
          {bins.length > 0 && (
            <List
              title="File under"
              keys={bins.map((c, i) => [<kbd>{i + 1}</kbd>, c.name])}
            />
          )}
          <List title="Sorting" keys={SORT} />
          <List title="Reading" keys={READING} />
          <List title="Anywhere" keys={APP} />
        </div>
      </dialog>
    </>
  );
}
