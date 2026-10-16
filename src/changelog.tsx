import { useSignal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";
import { Library, type About } from "./mygo";

/** The version whose changes the user last opened, kept to mark newer ones. */
const SEEN = "roost:seen-version";

type Block = { kind: "p"; text: string } | { kind: "ul"; items: string[] };
type Release = { version: string; date?: string; blocks: Block[] };

/**
 * Parses the Markdown CHANGELOG.md is written in: a `## version` heading per
 * release, then paragraphs and `-` lists whose items may wrap onto indented
 * lines. Everything before the first release is the file's title.
 */
export function parseChangelog(md: string): Release[] {
  const releases: Release[] = [];
  let para: string[] = [];
  let list: string[] | null = null;
  const flush = () => {
    const blocks = releases.at(-1)?.blocks;
    if (blocks && para.length) blocks.push({ kind: "p", text: para.join(" ") });
    if (blocks && list) blocks.push({ kind: "ul", items: list });
    para = [];
    list = null;
  };
  for (const line of md.split("\n")) {
    if (line.startsWith("## ")) {
      flush();
      // "## 1.2.0", or "## [1.2.0] - 2026-09-19" as in Keep a Changelog.
      const [, version, date] = line.slice(3).match(/^\[?([^\]\s]+)\]?(?:\s*-\s*(.+))?/) ?? [];
      releases.push({ version: version ?? line.slice(3).trim(), date: date?.trim(), blocks: [] });
    } else if (line.trim() === "") {
      flush();
    } else if (line.startsWith("- ")) {
      if (para.length) flush();
      (list ??= []).push(line.slice(2).trim());
    } else if (list && /^\s/.test(line)) {
      list[list.length - 1] += ` ${line.trim()}`;
    } else {
      if (list) flush();
      para.push(line.trim());
    }
  }
  flush();
  return releases;
}

/** Renders `code` spans; the rest is plain text. */
function inline(text: string) {
  return text.split(/`([^`]+)`/).map((part, i) => (i % 2 ? <code key={i}>{part}</code> : part));
}

/** The "What's new" button in the rail and the changelog it opens. */
export function WhatsNew() {
  const dialog = useRef<HTMLDialogElement>(null);
  const about = useSignal<About | null>(null);
  const unseen = useSignal(false);

  useEffect(() => {
    Library.about().then((a) => {
      about.value = a;
      const seen = localStorage.getItem(SEEN);
      // A fresh install has nothing to catch up on; an update does.
      if (seen === null) localStorage.setItem(SEEN, a.version);
      else unseen.value = seen !== a.version;
    });
  }, []);

  function open() {
    if (!about.value) return;
    localStorage.setItem(SEEN, about.value.version);
    unseen.value = false;
    dialog.current?.showModal();
  }

  const a = about.value;
  if (!a) return null;
  const releases = parseChangelog(a.changelog);
  return (
    <>
      <button class="add whats-new" onClick={open}>
        What’s new <span class="version">v{a.version}</span>
        {unseen.value && (
          <>
            <span class="new-dot" aria-hidden="true" />
            <span class="sr-only">, updated</span>
          </>
        )}
      </button>
      <dialog class="settings changelog" ref={dialog} aria-labelledby="changelog-title" onClick={(e) => e.target === dialog.current && dialog.current?.close()}>
        <div class="settings-body">
          <div class="sheet-head">
            <h2 id="changelog-title" class="sheet-kicker">
              What’s new
            </h2>
            <button class="close" onClick={() => dialog.current?.close()} aria-label="Close">
              ✕
            </button>
          </div>
          {releases.map((r) => (
            <section key={r.version} class="release" aria-labelledby={`release-${r.version}`}>
              <h3 id={`release-${r.version}`}>
                {r.version}
                {r.date && <span class="release-date">{r.date}</span>}
                {r.version === a.version && <span class="this-version">this version</span>}
              </h3>
              {r.blocks.map((b, i) =>
                b.kind === "p" ? (
                  <p key={i}>{inline(b.text)}</p>
                ) : (
                  <ul key={i}>
                    {b.items.map((item, j) => (
                      <li key={j}>{inline(item)}</li>
                    ))}
                  </ul>
                ),
              )}
            </section>
          ))}
        </div>
      </dialog>
    </>
  );
}
