import { useSignal } from "@preact/signals";
import { useRef } from "preact/hooks";
import { Library, type MediaUsage, type Settings } from "./mygo";
import { announce } from "./state";

const size = (bytes: number) =>
  bytes < 1e6 ? `${Math.round(bytes / 1e3)} KB` : bytes < 1e9 ? `${(bytes / 1e6).toFixed(0)} MB` : `${(bytes / 1e9).toFixed(1)} GB`;

/** The settings button in the rail and the dialog it opens. */
export function SettingsButton() {
  const dialog = useRef<HTMLDialogElement>(null);
  const settings = useSignal<Settings | null>(null);
  const usage = useSignal<MediaUsage | null>(null);

  async function open() {
    [settings.value, usage.value] = await Promise.all([Library.settings(), Library.mediaUsage()]);
    dialog.current?.showModal();
  }

  async function change(patch: Partial<Settings>) {
    if (!settings.value) return;
    const next = { ...settings.value, ...patch };
    settings.value = next;
    try {
      await Library.saveSettings(next);
    } catch (err) {
      announce(String(err));
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
            </>
          )}
        </div>
      </dialog>
    </>
  );
}
