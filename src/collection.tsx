import { useSignal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";
import { ink } from "./card";
import type { Collection } from "./mygo";
import { deleteCollection, errorText, HUES, updateCollection } from "./state";

const HUE_NAMES: Record<number, string> = {
  5: "Crimson",
  28: "Vermilion",
  50: "Orange",
  75: "Ochre",
  110: "Olive",
  145: "Green",
  195: "Teal",
  250: "Blue",
  280: "Violet",
  320: "Plum",
};

/**
 * Renames, recolours or deletes the collection on screen. A smart
 * collection's search can be changed too. Deleting asks first; the posts
 * themselves stay in roost.
 */
export function CollectionEditor({ c, onClose }: { c: Collection; onClose: () => void }) {
  const name = useSignal(c.name);
  const hue = useSignal(c.hue);
  const query = useSignal(c.query);
  const error = useSignal("");
  const confirming = useSignal(false);
  const nameField = useRef<HTMLInputElement>(null);
  const keep = useRef<HTMLButtonElement>(null);

  // Focus moves only when the editor opens or the delete question appears, never on a re-render.
  useEffect(() => nameField.current?.focus(), []);
  useEffect(() => {
    if (confirming.value) keep.current?.focus();
  }, [confirming.value]);

  async function save(e: Event) {
    e.preventDefault();
    try {
      await updateCollection({ ...c, name: name.value, hue: hue.value, query: query.value });
      onClose();
    } catch (err) {
      error.value = errorText(err);
    }
  }

  async function remove() {
    try {
      await deleteCollection(c);
      // The editor is gone with its collection; carry focus to the feed now on screen.
      document.getElementById("feed")?.focus({ preventScroll: true });
    } catch (err) {
      error.value = errorText(err);
      confirming.value = false;
    }
  }

  // Keep the collection's own hue on offer even if it is not one roost assigns.
  const hues = HUES.includes(c.hue) ? HUES : [c.hue, ...HUES];
  return (
    <form class="collection-editor" onSubmit={save} onKeyDown={(e) => {
        if (e.key !== "Escape") return;
        // Esc closes the editor only, not the post open beside it.
        e.stopPropagation();
        onClose();
      }} aria-label={`Edit ${c.name}`}>
      <label class="field">
        <span class="field-label">Name</span>
        <input
          value={name.value}
          onInput={(e) => ((name.value = e.currentTarget.value), (error.value = ""))}
          autoComplete="off"
          aria-invalid={error.value ? "true" : undefined}
          aria-describedby={error.value ? "collection-error" : undefined}
          ref={nameField}
        />
      </label>
      {c.query && (
        <label class="field">
          <span class="field-label">Search</span>
          <input value={query.value} onInput={(e) => (query.value = e.currentTarget.value)} autoComplete="off" spellcheck={false} />
        </label>
      )}
      <fieldset class="field swatches">
        <legend class="field-label">Colour</legend>
        {hues.map((h) => (
          <label key={h} class="swatch" style={{ "--hue": ink(h) }}>
            <input type="radio" name="hue" checked={hue.value === h} onChange={() => (hue.value = h)} />
            <span class="sr-only">{HUE_NAMES[h] ?? "Current colour"}</span>
          </label>
        ))}
      </fieldset>
      {error.value && (
        <p id="collection-error" class="field-error">
          {error.value}
        </p>
      )}
      {confirming.value ? (
        <div class="row confirm" role="group" aria-label="Confirm deletion">
          <p>Delete “{c.name}”? Its posts stay in roost.</p>
          <button type="button" class="press danger" onClick={remove}>
            Delete collection
          </button>
          <button type="button" class="press quiet" onClick={() => (confirming.value = false)} ref={keep}>
            Keep it
          </button>
        </div>
      ) : (
        <div class="row">
          <button class="press">Save changes</button>
          <button type="button" class="press quiet" onClick={onClose}>
            Cancel
          </button>
          <button type="button" class="press quiet delete" onClick={() => (confirming.value = true)}>
            Delete…
          </button>
        </div>
      )}
    </form>
  );
}
