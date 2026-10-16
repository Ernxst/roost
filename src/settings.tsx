import { useSignal } from "@preact/signals";
import { useRef } from "preact/hooks";
import { Library, type MediaUsage, type Result, type Settings } from "./mygo";
import { announce, count, fail } from "./state";

const size = (bytes: number) =>
  bytes < 1e6 ? `${Math.round(bytes / 1e3)} KB` : bytes < 1e9 ? `${(bytes / 1e6).toFixed(0)} MB` : `${(bytes / 1e9).toFixed(1)} GB`;

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

  async function change(patch: Partial<Settings>) {
    if (!settings.value) return;
    const next = { ...settings.value, ...patch };
    settings.value = next;
    try {
      await Library.saveSettings(next);
    } catch (err) {
      fail(err);
    }
  }

  const s = settings.value;
  return (
    <>
      <button class="add settings-button" onClick={open}>
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
          {s && (
            <>
              <section class="setting">
                <h3>Background sync</h3>
                <p class="hint">
                  roost checks X for new bookmarks while it is open, even with its window closed. It lives in the menu bar until you quit
                  it.
                </p>
                <label class="field">
                  <span>Check</span>
                  <select value={s.syncEvery} onChange={(e) => change({ syncEvery: Number(e.currentTarget.value) })}>
                    {syncChoices.map(([minutes, label]) => (
                      <option key={minutes} value={minutes}>
                        {label}
                      </option>
                    ))}
                  </select>
                </label>
                <label class="check">
                  <input type="checkbox" checked={s.notify} disabled={s.syncEvery === 0} onChange={(e) => change({ notify: e.currentTarget.checked })} />
                  <span>Tell me when new bookmarks arrive</span>
                </label>
              </section>
              <section class="setting">
                <h3>Offline media</h3>
                <p class="hint">
                  roost keeps a copy of every image, so posts still look right offline or after X takes them down.
                  {usage.value && ` ${usage.value.files.toLocaleString()} files, ${size(usage.value.bytes)}, are on this Mac.`}
                </p>
                <label class="check">
                  <input type="checkbox" checked={s.cacheVideos} onChange={(e) => change({ cacheVideos: e.currentTarget.checked })} />
                  <span>Keep videos too</span>
                </label>
              </section>
              <section class="setting">
                <h3>Markdown export</h3>
                <p class="hint">
                  One note per bookmark, with its thread, note, highlights, links and images, and one note per collection.
                  Point it at an Obsidian vault and the links between them just work.
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
