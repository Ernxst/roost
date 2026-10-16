import { useSignal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";
import { Library, type MediaUsage, type Result, type Settings } from "./mygo";
import { announce, count, errorText, fail, syncing, startSync } from "./state";

const size = (bytes: number) =>
  bytes < 1e6 ? `${Math.round(bytes / 1e3)} KB` : bytes < 1e9 ? `${(bytes / 1e6).toFixed(0)} MB` : bytes < 1e12 ? `${(bytes / 1e9).toFixed(1)} GB` : `${(bytes / 1e12).toFixed(1)} TB`;

const syncChoices: [number, string][] = [
  [0, "Only when I ask"],
  [15, "Every 15 minutes"],
  [30, "Every 30 minutes"],
  [60, "Every hour"],
  [180, "Every 3 hours"],
  [720, "Twice a day"],
];

/** The settings button in the rail and the dialog it opens. */
export function SettingsButton() {
  const dialog = useRef<HTMLDialogElement>(null);
  const settings = useSignal<Settings | null>(null);
  const usage = useSignal<MediaUsage | null>(null);
  const exporting = useSignal(false);
  const exported = useSignal<Result | null>(null);
  const saveError = useSignal("");
  const retryPatch = useSignal<Partial<Settings> | null>(null);
  const saving = useSignal(false);

  async function exportNotes(choose: boolean) {
    exporting.value = true;
    try {
      const res = await Library.export(choose);
      if (res.dir) {
        exported.value = res;
        settings.value = await Library.settings();
        announce(`Exported ${count(res.notes, "note")}`);
      }
    } catch (err) {
      fail(err);
    } finally {
      exporting.value = false;
    }
  }

  async function open() {
    try {
      [settings.value, usage.value] = await Promise.all([Library.settings(), Library.mediaUsage()]);
      dialog.current?.showModal();
    } catch (err) {
      fail(err);
    }
  }

  // ⌘, opens settings, as in every Mac app.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.metaKey && e.key === "," && !dialog.current?.open) {
        e.preventDefault();
        open();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  async function change(patch: Partial<Settings>) {
    const previous = settings.value;
    if (!previous || saving.value) return;
    const next = { ...previous, ...patch };
    settings.value = next;
    saving.value = true;
    saveError.value = "";
    retryPatch.value = null;
    try {
      await Library.saveSettings(next);
    } catch (err) {
      settings.value = previous;
      saveError.value = errorText(err);
      retryPatch.value = patch;
    } finally {
      saving.value = false;
    }
  }

  const s = settings.value;
  return (
    <>
      <button class="add settings-button" onClick={open} aria-keyshortcuts="Meta+,">
        Settings
      </button>
      <dialog class="settings" ref={dialog} aria-labelledby="settings-title" onClick={(e) => e.target === dialog.current && dialog.current?.close()}>
        <div class="settings-body">
          <div class="sheet-head">
            <h2 id="settings-title" class="sheet-kicker">
              Settings
            </h2>
            <button class="close" onClick={() => dialog.current?.close()} aria-label="Close">
              ✕
            </button>
          </div>
          {saveError.value && (
            <div class="settings-error" role="alert">
              <p>Unable to save this setting. Your previous value is restored. {saveError.value}</p>
              <button class="press quiet" disabled={saving.value} onClick={() => retryPatch.value && change(retryPatch.value)}>
                Try again
              </button>
            </div>
          )}
          {s && (
            <>
              <section class="setting">
                <h3>Background sync</h3>
                <p class="hint">
                  roost checks X for new bookmarks while it is open, even with its window closed. Once a week it reads every bookmark to
                  find bookmarks that are missing. They may have been unbookmarked or removed from X. roost lives in the menu bar until you quit it.
                </p>
                <label class="field">
                  <span>Check</span>
                  <select value={s.syncEvery} disabled={saving.value} onChange={(e) => change({ syncEvery: Number(e.currentTarget.value) })}>
                    {syncChoices.map(([minutes, label]) => (
                      <option key={minutes} value={minutes}>
                        {label}
                      </option>
                    ))}
                  </select>
                </label>
                <label class="check">
                  <input type="checkbox" checked={s.notify} disabled={saving.value || s.syncEvery === 0} onChange={(e) => change({ notify: e.currentTarget.checked })} />
                  <span>Tell me when bookmarks arrive or are missing</span>
                </label>
              </section>
              <section class="setting">
                <h3>Missing bookmarks</h3>
                <p class="hint">
                  A sync stops at the first bookmark roost already has. A full read finds bookmarks that are missing; they may have been
                  unbookmarked or removed from X. A large library takes a few minutes.
                </p>
                <div class="row">
                  <button class="press quiet" disabled={syncing.value} onClick={() => (dialog.current?.close(), startSync(true))}>
                    Read every bookmark
                  </button>
                </div>
              </section>
              <section class="setting">
                <h3>Offline media</h3>
                <p class="hint">
                  roost keeps a copy of every image, so posts still look right offline or after X takes them down.
                  {usage.value && ` ${count(usage.value.files, "file")}, ${size(usage.value.bytes)}, are on this Mac.`}
                </p>
                <label class="check">
                  <input type="checkbox" checked={s.cacheVideos} disabled={saving.value} onChange={(e) => change({ cacheVideos: e.currentTarget.checked })} />
                  <span>Keep videos too</span>
                </label>
              </section>
              <section class="setting">
                <h3>Markdown export</h3>
                <p class="hint">
                  One note per bookmark, with its thread, note, highlights, links and images, and one note per collection.
                  In an Obsidian vault, each bookmark links to its collections and each collection lists its bookmarks.
                </p>
                <div class="row">
                  {s.exportDir && (
                    <button class="press" disabled={exporting.value} onClick={() => exportNotes(false)}>
                      Export again
                    </button>
                  )}
                  <button class={s.exportDir ? "press quiet" : "press"} disabled={exporting.value} onClick={() => exportNotes(true)}>
                    {s.exportDir ? "Choose another folder…" : "Export to a folder…"}
                  </button>
                </div>
                {s.exportDir && (
                  <p class="link-status">
                    {exporting.value
                      ? "Writing notes…"
                      : exported.value
                        ? `Wrote ${count(exported.value.notes, "note")} and ${count(exported.value.media, "image")} to `
                        : "Last exported to "}
                    {!exporting.value && (
                      <button class="path" onClick={() => Library.reveal(exported.value?.dir ?? `${s.exportDir}/roost`)}>
                        {(exported.value?.dir ?? `${s.exportDir}/roost`).replace(/^\/Users\/[^/]+/, "~")}
                      </button>
                    )}
                  </p>
                )}
              </section>
            </>
          )}
        </div>
      </dialog>
    </>
  );
}
