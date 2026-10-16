import type { Author, Collection } from "./mygo";

/** A completion for the search word under the caret. */
export type Suggestion = {
  /** What replaces the word, e.g. `from:addyosmani `. */
  insert: string;
  /** What the list shows in mono, e.g. `from:addyosmani`. */
  label: string;
  /** The explanation beside it. */
  detail: string;
};

export type Completions = {
  /** Where the word being completed starts and ends in the input. */
  start: number;
  end: number;
  items: Suggestion[];
};

type Sources = { authors: Author[]; collections: Collection[]; today?: Date };

const KEYS: [string, string][] = [
  ["from:", "posts by an author"],
  ["has:", "media, links, notes and more"],
  ["in:", "in a collection"],
  ["is:", "gone, opened, unopened or set aside"],
  ["before:", "posted before a date"],
  ["after:", "posted after a date"],
];

const HAS: [string, string][] = [
  ["media", "any photo, video or GIF"],
  ["photo", "a photo"],
  ["video", "a video"],
  ["gif", "a GIF"],
  ["link", "a link"],
  ["article", "a link whose page roost kept"],
  ["quote", "a quoted post"],
  ["thread", "part of a thread"],
  ["note", "a note you wrote"],
  ["highlight", "a passage you highlighted"],
];

const IS: [string, string][] = [
  ["gone", "removed on X"],
  ["opened", "opened in roost"],
  ["unopened", "never opened in roost"],
  ["aside", "set aside while sorting"],
];

const LIMIT = 8;

/** The word under the caret runs back to the last space outside quotes. */
function wordAt(input: string, caret: number): { start: number; end: number } {
  let start = 0;
  let quoted = false;
  for (let i = 0; i < caret; i++) {
    if (input[i] === '"') quoted = !quoted;
    else if (input[i] === " " && !quoted) start = i + 1;
  }
  let end = caret;
  while (end < input.length && input[end] !== " ") end++;
  return { start, end };
}

const quote = (value: string) => (/\s/.test(value) ? `"${value}"` : value);
const day = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

function authorItem(neg: string, a: Author): Suggestion {
  const posts = `${a.count} ${a.count === 1 ? "post" : "posts"}`;
  return { insert: `${neg}from:${a.handle} `, label: `${neg}from:${a.handle}`, detail: a.name ? `${a.name}, ${posts}` : posts };
}

function collectionItem(neg: string, c: Collection): Suggestion {
  return { insert: `${neg}in:${quote(c.name)} `, label: `${neg}in:${quote(c.name)}`, detail: `${c.count} kept` };
}

/**
 * Completes the search word under the caret: an operator's name, an
 * operator's value (authors, collections, kinds, dates), or a plain word that
 * names an author or a collection. An empty word completes to nothing, so
 * the search hints show instead.
 */
export function complete(input: string, caret: number, { authors, collections, today = new Date() }: Sources): Completions {
  const { start, end } = wordAt(input, caret);
  const word = input.slice(start, caret);
  const neg = word.startsWith("-") ? "-" : "";
  const body = word.slice(neg.length);
  const colon = body.indexOf(":");
  const none = { start, end, items: [] };
  if (!body) return none;

  if (colon < 0) {
    const q = body.toLowerCase();
    const keys = KEYS.filter(([k]) => k.startsWith(q)).map(([k, detail]) => ({ insert: neg + k, label: neg + k, detail }));
    const byAuthor = authors
      .filter((a) => a.handle.toLowerCase().startsWith(q.replace(/^@/, "")) || a.name.toLowerCase().includes(q))
      .map((a) => authorItem(neg, a));
    const byCollection = collections.filter((c) => c.name.toLowerCase().includes(q)).map((c) => collectionItem(neg, c));
    return { start, end, items: [...keys, ...byCollection, ...byAuthor].slice(0, LIMIT) };
  }

  const key = body.slice(0, colon).toLowerCase();
  const value = body.slice(colon + 1).replace(/^"/, "").toLowerCase();
  const values = (pairs: [string, string][]) =>
    pairs
      .filter(([v]) => v.startsWith(value))
      .map(([v, detail]) => ({ insert: `${neg}${key}:${v} `, label: `${neg}${key}:${v}`, detail }));

  let items: Suggestion[];
  switch (key) {
    case "from":
      items = authors
        .filter((a) => a.handle.toLowerCase().startsWith(value.replace(/^@/, "")) || (value && a.name.toLowerCase().includes(value)))
        .map((a) => authorItem(neg, a));
      break;
    case "in":
      items = [
        ...collections.filter((c) => c.name.toLowerCase().startsWith(value)).map((c) => collectionItem(neg, c)),
        ...values([
          ["none", "in no collection"],
          ["any", "in at least one collection"],
        ]),
      ];
      break;
    case "has":
      items = values(HAS);
      break;
    case "is":
      items = values(IS);
      break;
    case "before":
    case "after": {
      const ago = (days: number, months = 0, years = 0) =>
        day(new Date(today.getFullYear() - years, today.getMonth() - months, today.getDate() - days));
      items = values([
        [ago(7), "a week ago"],
        [ago(0, 1), "a month ago"],
        [ago(0, 6), "six months ago"],
        [ago(0, 0, 1), "a year ago"],
      ]);
      break;
    }
    default:
      items = [];
  }
  return { start, end, items: items.slice(0, LIMIT) };
}

/** Puts a suggestion in place of the word it completes, with the caret after it. */
export function accept(input: string, c: Completions, s: Suggestion): { value: string; caret: number } {
  let after = input.slice(c.end);
  if (s.insert.endsWith(" ")) after = after.replace(/^ /, "");
  const value = input.slice(0, c.start) + s.insert + after;
  return { value, caret: c.start + s.insert.length };
}
