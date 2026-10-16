import { describe, expect, test } from "bun:test";
import { accept, complete } from "./autocomplete";
import type { Author, Collection } from "./mygo";

const authors: Author[] = [
  { handle: "addyosmani", name: "Addy Osmani", avatar: "", count: 12 },
  { handle: "fladdict", name: "深津 貴之", avatar: "", count: 1 },
];
const collections: Collection[] = [{ id: 1, name: "CSS & HTML", hue: 0, query: "", count: 52 }];
const sources = { authors, collections, today: new Date(2026, 9, 16) };
const labels = (input: string, caret = input.length) => complete(input, caret, sources).items.map((s) => s.label);

describe("complete", () => {
  test("offers nothing for an empty word, so the hints show", () => {
    expect(labels("")).toEqual([]);
    expect(labels("css ")).toEqual([]);
  });

  test("completes operator names, collections and authors from a plain word", () => {
    expect(labels("f")).toEqual(["from:", "from:fladdict"]);
    expect(labels("addy")).toEqual(["from:addyosmani"]);
    expect(labels("osmani")).toEqual(["from:addyosmani"]);
    expect(labels("css")).toEqual(['in:"CSS & HTML"']);
    expect(labels("dom")).toEqual(["domain:"]);
    expect(labels("la")).toEqual(["lang:"]);
    expect(labels("lik")).toEqual(["likes:"]);
  });

  test("completes values after an operator, keeping a leading minus", () => {
    expect(labels("from:ad")).toEqual(["from:addyosmani"]);
    expect(labels("has:v")).toEqual(["has:video"]);
    expect(labels("-in:")).toEqual(['-in:"CSS & HTML"', "-in:none", "-in:any"]);
    expect(labels("is:un")).toEqual(["is:unopened"]);
    expect(labels("before:")[0]).toBe("before:2026-10-09");
  });

  test("completes the word under the caret, not the last one", () => {
    expect(labels("has:v css", 5)).toEqual(["has:video"]);
  });

  test("treats a quoted value with spaces as one word", () => {
    expect(labels('in:"CSS &')).toEqual(['in:"CSS & HTML"']);
  });
});

describe("accept", () => {
  test("replaces the word and leaves the caret after it", () => {
    const c = complete("has:v css", 5, sources);
    expect(accept("has:v css", c, c.items[0])).toEqual({ value: "has:video css", caret: 10 });
  });

  test("leaves the caret after an operator name, ready for its value", () => {
    const c = complete("fr", 2, sources);
    expect(accept("fr", c, c.items[0])).toEqual({ value: "from:", caret: 5 });
  });
});
