import type { Author, Bookmark, Collection, ListResult, Stats, Tweet } from "./mygo";
import { fixtureMode } from "./fixture-mode";

const COLLECTIONS: Collection[] = [
  { id: 1, name: "Design", hue: 28, query: "", count: 42 },
  { id: 2, name: "Reading", hue: 145, query: "", count: 128 },
  { id: 3, name: "Research", hue: 250, query: "", count: 37 },
  { id: 4, name: "Q3 planning", hue: 75, query: "", count: 19 },
  { id: 5, name: "customer-feedback-from-enterprise-onboarding", hue: 320, query: "", count: 8 },
  { id: 6, name: "Accessibility", hue: 195, query: "", count: 64 },
  { id: 7, name: "Ideas", hue: 5, query: "", count: 91 },
  { id: 8, name: "Writing", hue: 110, query: "", count: 17 },
  { id: 9, name: "Product", hue: 280, query: "", count: 56 },
  { id: 10, name: "Language & culture", hue: 50, query: "", count: 13 },
  { id: 11, name: "Later", hue: 28, query: "", count: 1 },
  { id: 12, name: "Field notes", hue: 145, query: "", count: 22 },
];

const AUTHORS: Author[] = [
  { handle: "aleksandrawk123", name: "Aleksandra Wiśniewska-Kowalczyk", avatar: "", count: 1284 },
  { handle: "jo", name: "Jo", avatar: "", count: 1 },
  { handle: "j", name: "J", avatar: "https://example.com/roost-fixture-avatar.jpg", count: 12 },
  { handle: "olafuro", name: "Ólafur Darri Ólafsson", avatar: "", count: 9 },
  { handle: "dangngochan", name: "Đặng Thị Ngọc Hân", avatar: "", count: 8 },
  { handle: "wangxiuying", name: "王秀英", avatar: "", count: 7 },
  { handle: "nouralhuda", name: "نور الهدى عبد الرحمن", avatar: "", count: 6 },
  { handle: "priyaf", name: "👩🏽‍💻 Priya", avatar: "", count: 5 },
];

const now = Date.now();
const iso = (delta: number) => new Date(now + delta).toISOString();
const firstSeenAt = iso(-1000 * 60 * 60 * 24 * 90);
const longUrl = "https://example.com/workspaces/roost/projects/q3-launch/docs/9f8e7d6c5b4a?tab=comments&filter=unresolved";
const longNote = [
  "Keep this example because the small interface decision changes how the whole workflow feels.",
  "The useful detail is the context around the quote, which is easy to lose when the card only shows a short preview.",
  "When we revisit this, compare the original source with the note and keep the author attribution intact.",
  "A longer field note is realistic after a research session, especially when someone pastes several observations at once.",
].join(" ").repeat(5).slice(0, 2000);

function tweet(
  id: string,
  author: Pick<Author, "handle" | "name" | "avatar">,
  text: string,
  createdAt = iso(-1000 * 60 * 60 * 24 * 12),
  extra: Partial<Tweet> = {},
): Tweet {
  return {
    id,
    url: `https://x.com/${author.handle}/status/${id}`,
    authorHandle: author.handle,
    authorName: author.name,
    authorAvatar: author.avatar,
    text,
    createdAt,
    media: [],
    links: [],
    ...extra,
  };
}

function bookmark(
  post: Tweet,
  extra: Partial<Bookmark> = {},
): Bookmark {
  return {
    tweet: post,
    note: "",
    collections: [],
    firstSeenAt,
    removedAt: null,
    openedAt: null,
    asideAt: null,
    threadLength: 1,
    ...extra,
  };
}

const longName = AUTHORS[0]!;
const shortName = AUTHORS[1]!;
const oneLetter = AUTHORS[2]!;
const vietnamese = AUTHORS[4]!;
const cjk = AUTHORS[5]!;
const rtl = AUTHORS[6]!;
const emoji = AUTHORS[7]!;

const edgeCases: Bookmark[] = [
  bookmark(tweet("fixture-worst-01", longName, "A saved post from Aleksandra Wiśniewska-Kowalczyk, with the author name long enough to compete with the handle and timestamp.")),
  bookmark(tweet("fixture-worst-02", shortName, "Jo shared a short name and a post with exactly one line."), { collections: [1] }),
  bookmark(tweet("fixture-worst-03", oneLetter, "J is a one-letter display name. The avatar URL deliberately points at a missing example.com image.", iso(-1000 * 60 * 60 * 2))),
  bookmark(tweet("fixture-worst-04", vietnamese, "Đặng Thị Ngọc Hân wrote about typography in Vietnamese, with stacked marks above and below the line."), { collections: [2, 3] }),
  bookmark(tweet("fixture-worst-05", cjk, "王秀英保存了这条没有空格的中文帖子，用来检查姓名、换行和卡片高度。"), { collections: [4] }),
  bookmark(tweet("fixture-worst-06", rtl, "نور الهدى عبد الرحمن كتبت ملاحظة قصيرة عن تجربة القراءة باللغة العربية."), { collections: [5] }),
  bookmark(tweet("fixture-worst-07", emoji, "👩🏽‍💻 Priya posted about the small details that make a keyboard shortcut easier to discover."), { collections: [6, 7] }),
  bookmark(tweet("fixture-worst-08", AUTHORS[3]!, `The full source is linked here: ${longUrl}`), { collections: [8] }),
  bookmark(tweet("fixture-worst-09", { handle: "missingavatar", name: "Missing Avatar", avatar: "" }, "No avatar was supplied for this author, so the blank placeholder should keep its size."), { collections: [9] }),
  bookmark(tweet("fixture-worst-10", { handle: "brokenimage", name: "Broken Image", avatar: "https://example.com/roost-fixture-avatar.jpg" }, "A very wide image should keep a useful shape instead of becoming a strip.", undefined, {
    media: [{ kind: "photo", thumb: "https://example.com/roost-fixture-panorama.jpg", width: 4000, height: 200 }],
  })),
  bookmark(tweet("fixture-worst-11", AUTHORS[1]!, "The card's note preview has 2,000 characters. It is a realistic pasted research note with several complete sentences."), { note: longNote, collections: [10, 11, 12] }),
  bookmark(tweet("fixture-worst-12", longName, "A post with twelve collection stamps checks how a full set wraps across a narrow card."), { collections: COLLECTIONS.map((c) => c.id) }),
  bookmark(tweet("fixture-worst-13", oneLetter, "This post is dated three days in the future to check the relative timestamp."), { tweet: tweet("fixture-worst-13", oneLetter, "This post is dated three days in the future to check the relative timestamp.", iso(1000 * 60 * 60 * 24 * 3)) }),
  bookmark(tweet("fixture-worst-14", vietnamese, ""), { collections: [1] }),
  bookmark(tweet("fixture-worst-15", cjk, "Literal markup stays text: &lt;b&gt;bold-looking&lt;/b&gt;, &amp;, and **markdown**.\nA second line keeps its line break."), {
    tweet: tweet("fixture-worst-15", cjk, "A quoted post adds a second long author line and a long body to the same card.", undefined, {
      quoted: tweet("fixture-quote-15", rtl, "A long quoted passage with enough words to check how nested text wraps when both the original author and the quote have long content."),
    }),
  }),
  bookmark(tweet("fixture-worst-16", AUTHORS[2]!, "A post with a missing optional note, no media, and no collection should leave no placeholder gap."), { removedAt: iso(-1000 * 60 * 60 * 24 * 4), threadLength: 18 }),
];

const fillerAuthors = [longName, shortName, oneLetter, vietnamese, cjk, rtl, emoji];
const worstItems = [
  ...edgeCases,
  ...Array.from({ length: 1284 - edgeCases.length }, (_, index) => {
    const n = index + edgeCases.length + 1;
    const author = fillerAuthors[index % fillerAuthors.length]!;
    return bookmark(
      tweet(`fixture-worst-${String(n).padStart(4, "0")}`, author, `A saved field note about product decisions, careful research, and details worth returning to (${n}).`, iso(-1000 * 60 * 60 * 24 * (13 + (index % 900)))),
      { collections: n % 3 === 0 ? [((n - 1) % COLLECTIONS.length) + 1] : [] },
    );
  }),
];

const oneItem = bookmark(
  tweet("fixture-one", shortName, "One saved post. One item in every collection. One is also the edge case for singular labels."),
  { collections: COLLECTIONS.map((c) => c.id), removedAt: iso(-1000 * 60 * 60 * 24), threadLength: 1 },
);

function generatedItems(count: number, prefix: string): Bookmark[] {
  return Array.from({ length: count }, (_, index) => {
    const n = index + 1;
    const author = fillerAuthors[index % fillerAuthors.length]!;
    return bookmark(
      tweet(`${prefix}-${String(n).padStart(4, "0")}`, author, `A saved post about notes, reading, and keeping useful ideas close (${n}).`, iso(-1000 * 60 * 60 * 24 * (1 + (index % 900)))),
      { collections: n % 4 === 0 ? [((n - 1) % COLLECTIONS.length) + 1] : [] },
    );
  });
}

export function fixtureList(offset: number, limit: number): ListResult {
  const source = fixtureMode === "worst" ? worstItems : fixtureMode === "one" ? [oneItem] : fixtureMode === "thousand" ? thousandItems : [];
  return { items: source.slice(offset, offset + limit), total: source.length };
}

const thousandItems = generatedItems(1000, "fixture-thousand");

export function fixtureMeta(): { stats: Stats; collections: Collection[]; authors: Author[] } {
  const total = fixtureMode === "worst" ? worstItems.length : fixtureMode === "one" ? 1 : fixtureMode === "thousand" ? thousandItems.length : 0;
  const unit = fixtureMode === "one" ? 1 : undefined;
  const fixtureCollections = fixtureMode === "empty" ? [] : COLLECTIONS;
  const fixtureAuthors = fixtureMode === "empty" ? [] : fixtureMode === "one" ? [shortName] : AUTHORS;
  return {
    stats: {
      total,
      removed: unit ?? (fixtureMode === "worst" ? 1 : 0),
      unsorted: unit ?? (fixtureMode === "worst" ? 1240 : Math.min(total, 700)),
      toSort: unit ?? (fixtureMode === "worst" ? 999 : Math.min(total, 300)),
      lastSync: {
        full: true,
        startedAt: iso(-1000 * 60 * 60 * 3),
        finishedAt: iso(-1000 * 60 * 60 * 2),
        added: fixtureMode === "worst" ? 1284 : total,
        seen: total,
        removed: fixtureMode === "worst" ? 1 : unit ?? 0,
        reachedEnd: true,
        error: "",
      },
    },
    collections: fixtureCollections.map((c) => ({ ...c, count: unit ?? c.count })),
    authors: fixtureAuthors.map((a) => ({ ...a, count: unit ?? a.count })),
  };
}

