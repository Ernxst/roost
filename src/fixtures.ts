import changelog from "../CHANGELOG.md?raw";
import { emit } from "./fixture-runtime";
import { fixtureMode } from "./fixture-mode";
import type {
  AuthorSummary,
  Author,
  Bookmark,
  Collection,
  Duplicates,
  Highlight,
  ListResult,
  Media,
  Page,
  Progress,
  Query,
  Revisit,
  SavedLink,
  Stats,
  Tweet,
} from "./mygo";

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
  { id: 13, name: "Mum's \"recipes to try\"", hue: 250, query: "", count: 4 },
  { id: 14, name: "Barrierefreiheitsanforderungen", hue: 75, query: "", count: 2 },
  { id: 15, name: "Long reads I haven't opened from people I follow", hue: 320, query: "has:link -is:opened lang:en likes:>500 -domain:medium.com", count: 1284 },
];

const AUTHORS: Author[] = [
  { handle: "aleksandrawk123", name: "Aleksandra Wiśniewska-Kowalczyk", avatar: "", count: 1284 },
  { handle: "jo", name: "Jo", avatar: "", count: 1 },
  { handle: "j", name: "J", avatar: "https://pbs.twimg.com/profile_images/fixture/missing_normal.jpg", count: 12 },
  { handle: "olafuro", name: "Ólafur Darri Ólafsson", avatar: "", count: 9 },
  { handle: "dangngochan", name: "Đặng Thị Ngọc Hân", avatar: "", count: 8 },
  { handle: "wangxiuying", name: "王秀英", avatar: "", count: 7 },
  { handle: "nouralhuda", name: "نور الهدى عبد الرحمن", avatar: "", count: 6 },
  { handle: "priyaf", name: "👩🏽‍💻 Priya", avatar: "", count: 5 },
  // X's longest display name is 50 characters and its longest handle 15.
  { handle: "fitzgeraldmontg", name: "Dr. Bartholomew Fitzgerald-Montgomery III, PhD 🇬🇧", avatar: "", count: 3 },
  { handle: "dailydesignlink", name: "[BOT] Daily Design Links", avatar: "", count: 2 },
  // Posts read from a thread or an import can arrive without a display name.
  { handle: "nameless_reader", name: "", avatar: "", count: 1 },
];

const now = Date.now();
const DAY = 1000 * 60 * 60 * 24;
const iso = (delta: number) => new Date(now + delta).toISOString();
const firstSeenAt = iso(-90 * DAY);
const longUrl = "https://example.com/workspaces/roost/projects/q3-launch/docs/9f8e7d6c5b4a?tab=comments&filter=unresolved";
const longNote = [
  "Keep this example because the small interface decision changes how the whole workflow feels.",
  "The useful detail is the context around the quote, which is easy to lose when the card only shows a short preview.",
  "When we revisit this, compare the original source with the note and keep the author attribution intact.",
  "A longer field note is realistic after a research session, especially when someone pastes several observations at once.",
].join(" ").repeat(5).slice(0, 2000);

// X Premium posts run to 25,000 characters; essays of a few thousand are common among bookmarks.
const longPost = [
  "I spent the last six months rebuilding our onboarding, and the single biggest lesson was that the empty state is the product for most people. Here is everything we learned, in order.",
  "1. Nobody reads the welcome modal. We measured it: median time on the first screen was 1.4 seconds, which is about how long it takes to find the close button. We replaced it with a checklist that lives in the sidebar and shrinks as you finish things.",
  "2. Demo data lies. Our designs were all built against a workspace with twelve members, short names and perfect avatars. Real workspaces have 1,284 members, people called Jo, people whose names wrap onto three lines, and half of them never uploaded a photo.",
  "3. Every count needs a plural rule and a separator. \"1 members\" shipped to production twice. So did \"12345 tasks\".",
  "4. The longest translated label decides your button width, not the English one. German and Finnish broke our toolbar in a week.",
  "5. Errors are copy too. A raw stack trace in a toast is an apology nobody can read.",
].join("\n\n").repeat(3);

function tweet(
  id: string,
  author: Pick<Author, "handle" | "name" | "avatar">,
  text: string,
  createdAt = iso(-12 * DAY),
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

function bookmark(post: Tweet, extra: Partial<Bookmark> = {}): Bookmark {
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

const photo = (name: string, width: number, height: number, description?: string): Media => ({
  kind: "photo",
  thumb: `https://pbs.twimg.com/media/${name}.jpg`,
  width,
  height,
  description,
});
const video = (name: string, width: number, height: number): Media => ({
  kind: "video",
  thumb: `https://pbs.twimg.com/ext_tw_video_thumb/${name}.jpg`,
  video: `https://video.twimg.com/ext_tw_video/${name}.mp4`,
  width,
  height,
});
const gif = (name: string, width: number, height: number): Media => ({
  kind: "animated_gif",
  thumb: `https://pbs.twimg.com/tweet_video_thumb/${name}.jpg`,
  video: `https://video.twimg.com/tweet_video/${name}.mp4`,
  width,
  height,
});

const [longName, shortName, oneLetter, icelandic, vietnamese, cjk, rtl, emoji, maxName, bot, nameless] = AUTHORS as [
  Author, Author, Author, Author, Author, Author, Author, Author, Author, Author, Author,
];

const pdfLink = {
  short: "https://t.co/q3launchpdf",
  expanded: "https://northwind-industries-holdings.example.com/files/Q3-launch-plan_v12_FINAL_final-reviewed-by-legal.pdf",
  display: "northwind-industries-holdings.example.com/files/Q3-l…",
};
const deadLink = { short: "https://t.co/deadlink01", expanded: longUrl, display: "example.com/workspaces/roost/…" };
const essayLink = {
  short: "https://t.co/essaylink1",
  expanded: "https://www.longreads.example.org/2026/10/the-empty-state-is-the-product",
  display: "longreads.example.org/2026/10/the-e…",
};

const edgeCases: Bookmark[] = [
  bookmark(tweet("fixture-worst-01", longName, "A saved post from Aleksandra Wiśniewska-Kowalczyk, with the author name long enough to compete with the handle and timestamp.")),
  bookmark(tweet("fixture-worst-02", shortName, "Jo shared a short name and a post with exactly one line."), { collections: [1] }),
  bookmark(tweet("fixture-worst-03", maxName, longPost, iso(-3 * DAY)), { collections: [2, 15] }),
  bookmark(tweet("fixture-worst-04", vietnamese, "Đặng Thị Ngọc Hân wrote about typography in Vietnamese, with stacked marks above and below the line."), { collections: [2, 3] }),
  bookmark(tweet("fixture-worst-05", cjk, "王秀英保存了这条没有空格的中文帖子，用来检查姓名、换行和卡片高度。"), { collections: [4] }),
  bookmark(tweet("fixture-worst-06", rtl, "نور الهدى عبد الرحمن كتبت ملاحظة قصيرة عن تجربة القراءة باللغة العربية."), { collections: [5] }),
  bookmark(tweet("fixture-worst-07", emoji, "👩🏽‍💻 Priya posted about the small details that make a keyboard shortcut easier to discover."), { collections: [6, 7] }),
  bookmark(tweet("fixture-worst-08", icelandic, `The full plan, and the essay that started it: https://t.co/q3launchpdf https://t.co/essaylink1 (the old doc moved: https://t.co/deadlink01)`, undefined, {
    links: [pdfLink, essayLink, deadLink],
  }), { collections: [8] }),
  bookmark(tweet("fixture-worst-09", nameless, "No display name came with this post, so the avatar has no letter and the byline only a handle."), { collections: [9] }),
  bookmark(tweet("fixture-worst-10", bot, "Four screenshots of the same settings page at different widths.", undefined, {
    media: [photo("fixture-tall", 1170, 2532), photo("fixture-wide", 4000, 200), photo("fixture-square", 1080, 1080), photo("fixture-nodims", 0, 0)],
  }), { collections: [13] }),
  bookmark(tweet("fixture-worst-11", shortName, "The card's note preview has 2,000 characters. It is a realistic pasted research note with several complete sentences."), { note: longNote, collections: [10, 11, 12] }),
  bookmark(tweet("fixture-worst-12", longName, "A post with fifteen collection stamps checks how a full set wraps across a narrow card.", undefined, {
    media: [gif("fixture-gif", 498, 280)],
  }), { collections: COLLECTIONS.map((c) => c.id) }),
  bookmark(tweet("fixture-worst-13", oneLetter, "This post is dated three days in the future to check the relative timestamp.", iso(3 * DAY))),
  bookmark(tweet("fixture-worst-14", vietnamese, "", undefined, { media: [video("fixture-video", 720, 1280)] }), { collections: [1] }),
  bookmark(tweet("fixture-worst-15", cjk, "A quoted post adds a second long author line and a long body to the same card.", undefined, {
    quoted: tweet("fixture-quote-15", rtl, "A long quoted passage with enough words to check how nested text wraps when both the original author and the quote have long content."),
  })),
  bookmark(tweet("fixture-worst-16", oneLetter, "A post with a missing optional note, no media, and no collection should leave no placeholder gap.", undefined, {
    media: [photo("fixture-gone", 1600, 900)],
  }), { removedAt: iso(-4 * DAY), threadLength: 18 }),
  bookmark(tweet("fixture-worst-17", emoji, "Set aside from sorting: it waits outside the queue."), { asideAt: iso(-DAY) }),
];

const fillerAuthors = [longName, shortName, oneLetter, vietnamese, cjk, rtl, emoji, maxName, bot];

function generated(count: number, prefix: string, from: number, withMedia: boolean): Bookmark[] {
  return Array.from({ length: count }, (_, index) => {
    const n = index + from;
    const author = fillerAuthors[index % fillerAuthors.length]!;
    const media = withMedia || n % 5 === 0 ? [n % 7 === 0 ? video(`${prefix}-${n}`, 1280, 720) : photo(`${prefix}-${n}`, 1200, 800 + (n % 9) * 150)] : [];
    return bookmark(
      tweet(`${prefix}-${String(n).padStart(4, "0")}`, author, `A saved field note about product decisions, careful research, and details worth returning to (${n}).`, iso(-(13 + (index % 900)) * DAY), { media }),
      { collections: n % 3 === 0 ? [((n - 1) % 12) + 1] : [] },
    );
  });
}

const oneItem = bookmark(
  tweet("fixture-one", shortName, "One saved post. One item in every collection. One is also the edge case for singular labels.", undefined, {
    media: [photo("fixture-one", 1200, 800)],
    links: [essayLink],
  }),
  { collections: COLLECTIONS.map((c) => c.id), removedAt: iso(-DAY) },
);

const library: Bookmark[] =
  fixtureMode === "worst"
    ? [...edgeCases, ...generated(1284 - edgeCases.length, "fixture-worst", edgeCases.length + 1, false)]
    : fixtureMode === "one"
      ? [oneItem]
      : fixtureMode === "thousand"
        ? generated(1000, "fixture-thousand", 1, true)
        : [];

const collections: Collection[] = fixtureMode === "empty" ? [] : COLLECTIONS.map((c) => ({ ...c, count: library.filter((b) => b.collections.includes(c.id)).length }));

const fillerNames = ["Mira Okonkwo", "Tomasz Zieliński", "Sakura Yamamoto-Brennan", "Kwame Mensah", "Inès Lefèvre", "Mateus Araújo", "Siobhán Ní Bhriain", "Rafael Domínguez"];
const authors: Author[] =
  fixtureMode === "empty"
    ? []
    : fixtureMode === "one"
      ? [{ ...shortName, count: 1 }]
      : [
          ...AUTHORS,
          ...Array.from({ length: (fixtureMode === "thousand" ? 1000 : 1284) - AUTHORS.length }, (_, i) => ({
            handle: `reader${String(i + 1).padStart(4, "0")}`,
            name: `${fillerNames[i % fillerNames.length]} ${i + 1}`,
            avatar: "",
            count: Math.max(1, 900 - i),
          })),
        ];

function matches(b: Bookmark, q: Query) {
  if (q.removed ? !b.removedAt : b.removedAt) return false;
  if (q.collection > 0 && !b.collections.includes(q.collection)) return false;
  if (q.collection === -1 && b.collections.length) return false;
  if (q.author && b.tweet.authorHandle.toLowerCase() !== q.author.toLowerCase()) return false;
  if (q.mediaOnly && !b.tweet.media.length) return false;
  const words = q.search.split(/\s+/).filter(Boolean);
  if (words.includes("-is:aside") && b.asideAt) return false;
  // Search operators other than the sorting queue's are not modelled.
  const text = `${b.tweet.text} ${b.tweet.authorName} ${b.note}`.toLowerCase();
  return words.filter((w) => !w.includes(":")).every((w) => text.includes(w.toLowerCase()));
}

function stats(): Stats {
  const kept = library.filter((b) => !b.removedAt);
  const unsorted = kept.filter((b) => b.collections.length === 0);
  return {
    total: kept.length,
    removed: library.length - kept.length,
    unsorted: unsorted.length,
    toSort: unsorted.filter((b) => !b.asideAt).length,
    lastSync: fixtureMode === "empty"
      ? null
      : {
          full: true,
          startedAt: iso(-3 * 60 * 60 * 1000),
          finishedAt: iso(-2 * 60 * 60 * 1000),
          added: library.length,
          seen: library.length,
          removed: library.length - kept.length,
          reachedEnd: true,
          error: "",
        },
  };
}

function summary(handle: string): AuthorSummary {
  const a = authors.find((x) => x.handle.toLowerCase() === handle.toLowerCase()) ?? { handle, name: "", avatar: "", count: 0 };
  const theirs = library.filter((b) => b.tweet.authorHandle === a.handle);
  const filed = new Map<number, number>();
  for (const b of theirs) for (const id of b.collections) filed.set(id, (filed.get(id) ?? 0) + 1);
  return {
    ...a,
    gone: theirs.filter((b) => b.removedAt).length,
    collections: [...filed].map(([id, count]) => ({ id, count })),
  };
}

const pages: Record<string, Page> = {
  [pdfLink.expanded]: {
    url: pdfLink.expanded,
    finalUrl: pdfLink.expanded,
    title: "Q3-launch-plan_v12_FINAL_final-reviewed-by-legal.pdf",
    site: "northwind-industries-holdings.example.com",
    byline: "",
    excerpt: "",
    error: "",
  },
  [essayLink.expanded]: {
    url: essayLink.expanded,
    finalUrl: `${essayLink.expanded}?utm_source=x&utm_medium=social&utm_campaign=autumn-2026-newsletter-relaunch`,
    title: "The empty state is the product: what six months of onboarding research taught a team that thought it already knew",
    site: "The Long Reads Quarterly — Essays on Design, Software & the People Who Make Them",
    byline: "By Aleksandra Wiśniewska-Kowalczyk, Dr. Bartholomew Fitzgerald-Montgomery III and Jo",
    excerpt: longPost.slice(0, 600),
    text: [
      ...longPost.split("\n\n").slice(0, 6),
      "The raw export we worked from is still up at https://northwind-industries-holdings.example.com/research/onboarding/2026-q2/exports/sessions-anonymised-v3-final.csv?download=1&token=9f8e7d6c5b4a3f2e1d0c",
      "我们在中文界面里也做了同样的测试，发现没有空格的长句子在窄屏上更容易出问题。",
    ].join("\n\n"),
  },
  [deadLink.expanded]: {
    url: deadLink.expanded,
    finalUrl: deadLink.expanded,
    title: "",
    site: "example.com",
    byline: "",
    excerpt: "",
    error: `Get "${deadLink.expanded}": dial tcp: lookup example.com on 192.168.1.1:53: no such host`,
  },
};

function links(id: string): SavedLink[] {
  const b = library.find((x) => x.tweet.id === id);
  return (b?.tweet.links ?? []).map((l) => {
    const page = pages[l.expanded] ?? null;
    return { ...l, page, hasText: !!page?.text };
  });
}

const longQuote = longPost.split("\n\n").slice(1, 3).join(" ");
const highlights: Highlight[] =
  fixtureMode === "empty"
    ? []
    : fixtureMode === "one"
      ? [{ id: 1, tweetId: oneItem.tweet.id, source: "", quote: "One saved post.", note: "", createdAt: iso(-DAY), tweet: oneItem.tweet }]
      : [
          { id: 1, tweetId: "fixture-worst-08", source: essayLink.expanded, quote: longQuote, note: longNote.slice(0, 600), createdAt: iso(-2 * 60 * 1000) },
          { id: 2, tweetId: "fixture-worst-05", source: "", quote: "王秀英保存了这条没有空格的中文帖子，用来检查姓名、换行和卡片高度。", note: "", createdAt: iso(-3 * 60 * 60 * 1000) },
          { id: 3, tweetId: "fixture-worst-06", source: "", quote: "تجربة القراءة باللغة العربية", note: "ملاحظة قصيرة", createdAt: iso(-2 * DAY) },
          { id: 4, tweetId: "fixture-worst-03", source: "", quote: "Demo data lies.", note: "", createdAt: iso(-3 * DAY) },
          { id: 5, tweetId: "fixture-gone-from-x", source: "", quote: "A highlight whose post roost no longer has on record.", note: "", createdAt: iso(-40 * DAY) },
          { id: 6, tweetId: "fixture-worst-08", source: "https://northwind-industries-holdings.example.com/research/onboarding", quote: "https://northwind-industries-holdings.example.com/research/onboarding/2026-q2/exports/sessions-anonymised-v3-final.csv", note: "", createdAt: iso(-41 * DAY) },
          ...Array.from({ length: fixtureMode === "thousand" ? 994 : 0 }, (_, i) => ({
            id: 7 + i,
            tweetId: library[i % library.length]!.tweet.id,
            source: "",
            quote: `A kept passage about careful research (${i + 1}).`,
            note: i % 4 === 0 ? "Worth a second look." : "",
            createdAt: iso(-(i + 1) * DAY),
          })),
        ];
const byId = new Map(library.map((b) => [b.tweet.id, b.tweet]));
for (const h of highlights) h.tweet ??= byId.get(h.tweetId);

function thread(id: string): Tweet[] {
  const b = library.find((x) => x.tweet.id === id);
  if (!b || b.threadLength < 2) return [];
  return Array.from({ length: b.threadLength }, (_, i) =>
    i === 3 ? b.tweet : tweet(`${id}-thread-${i}`, oneLetter, `Part ${i + 1} of the thread, continuing the argument with one more example.`, iso(-(12 * DAY) + i * 60 * 1000)),
  );
}

const withMedia = library.filter((b) => b.tweet.media.length);
const duplicates: Duplicates[] =
  fixtureMode === "worst"
    ? [
        { reason: "link", key: `${longUrl}&utm_source=x`, bookmarks: [edgeCases[7]!, edgeCases[11]!, edgeCases[0]!] },
        { reason: "image", key: "https://pbs.twimg.com/media/fixture-tall.jpg", bookmarks: [edgeCases[9]!, edgeCases[2]!] },
        { reason: "post", key: "fixture-worst-15", bookmarks: [edgeCases[14]!, edgeCases[13]!] },
      ]
    : fixtureMode === "thousand"
      ? Array.from({ length: 300 }, (_, i) => ({ reason: "post", key: `set-${i}`, bookmarks: [library[i * 2]!, library[i * 2 + 1]!] }))
      : [];

function revisit(): Revisit {
  if (fixtureMode === "empty") return { onThisDay: [], unopened: [], random: [] };
  if (fixtureMode === "one") return { onThisDay: [], unopened: [oneItem], random: [oneItem] };
  const shuffled = [...library].sort(() => Math.random() - 0.5);
  return { onThisDay: edgeCases.slice(0, 6), unopened: edgeCases.slice(6, 12), random: shuffled.slice(0, 6) };
}

/** Acts out a sync that reads a while, then fails with a message as long as Go writes them. */
function sync(full: boolean) {
  const at = (ms: number, p: Progress) => setTimeout(() => emit("syncProgress", p), ms);
  at(300, { phase: "reading", full, seen: 1240, added: 0, removed: 0, message: "" });
  at(1500, { phase: "reading", full, seen: 12_480, added: 1284, removed: 0, message: "" });
  at(3000, {
    phase: "failed",
    full,
    seen: 12_480,
    added: 1284,
    removed: 0,
    message: `reading bookmarks: Get "https://x.com/i/api/graphql/QUjXply7fA7fk05FRyajEg/Bookmarks?variables=%7B%22count%22%3A20%2C%22cursor%22%3A%22HBaAgLydmdz9qjMAAA%3D%3D%22%7D&features=%7B%22graphql_timeline_v2_bookmark_timeline%22%3Atrue%7D": net/http: TLS handshake timeout`,
  });
}

/** Answers a Library call as Go would, from the fixture in the URL. Changes are accepted and forgotten. */
export function answer(method: string, args: unknown[]): unknown {
  const [first, second] = args;
  switch (method) {
    case "Library.List": {
      const q = first as Query;
      const found = library.filter((b) => matches(b, q));
      return { items: found.slice(q.offset, q.offset + q.limit), total: found.length } satisfies ListResult;
    }
    case "Library.Stats":
      return stats();
    case "Library.Collections":
      return collections;
    case "Library.Authors":
      return (first as number) < 0 ? authors : authors.slice(0, first as number);
    case "Library.AuthorSummary":
      return summary(first as string);
    case "Library.Get":
      return library.find((b) => b.tweet.id === first) ?? Promise.reject(new Error("no such bookmark"));
    case "Library.Revisit":
      return revisit();
    case "Library.AllHighlights":
      return highlights;
    case "Library.Highlights":
      return highlights.filter((h) => h.tweetId === first);
    case "Library.Wall":
      return withMedia;
    case "Library.Duplicates":
      return duplicates;
    case "Library.Links":
      return links(first as string);
    case "Library.Page":
      return pages[first as string] ?? null;
    case "Library.Thread":
    case "Library.LoadThread":
      return thread(first as string);
    case "Library.Suggest":
      return [15, 5, 14].map((collectionId, i) => ({ collectionId, score: 1 - i / 10 }));
    case "Library.Settings":
      return {
        cacheVideos: false,
        syncEvery: 30,
        notify: true,
        exportDir: fixtureMode === "empty" ? "" : "/Users/aleksandra/Library/Mobile Documents/iCloud~md~obsidian/Documents/Aleksandra-Research-Vault-2026",
      };
    case "Library.MediaUsage":
      return fixtureMode === "one" ? { files: 1, bytes: 48_213 } : fixtureMode === "empty" ? { files: 0, bytes: 0 } : { files: 48_213, bytes: 1_234_567_890_123 };
    case "Library.About":
      return { version: "0.2.0", changelog };
    case "Library.Sync":
      return sync(first as boolean);
    case "Library.CreateCollection":
      return { id: 100 + Math.floor(Math.random() * 1000), name: first, hue: second, query: args[2], count: 0 } as Collection;
    case "Library.AddHighlight":
      return { id: 10_000 + Math.floor(Math.random() * 1000), tweetId: first, source: second, quote: args[2], note: "", createdAt: new Date().toISOString() } as Highlight;
    case "Library.Export":
      return { dir: "/Users/aleksandra/Library/Mobile Documents/iCloud~md~obsidian/Documents/Aleksandra-Research-Vault-2026/roost", notes: library.length, media: 48_213 };
    case "Library.ShareCollection":
      return "/Users/aleksandra/Desktop/customer-feedback-from-enterprise-onboarding.md";
    case "Library.Import":
      return 0;
    case "Library.Open":
    case "Library.Reveal":
    case "Library.MarkOpened":
    case "Library.SetAside":
    case "Library.SetInCollection":
    case "Library.SetNote":
    case "Library.SetHighlightNote":
    case "Library.DeleteHighlight":
    case "Library.RestoreHighlight":
    case "Library.UpdateCollection":
    case "Library.DeleteCollection":
    case "Library.SaveSettings":
      return undefined;
    default:
      return Promise.reject(new Error(`${method} has no fixture`));
  }
}
