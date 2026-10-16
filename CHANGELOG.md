# Changelog

## 0.2.1

Sync reads your whole library again, roost has a new icon, and long names,
long posts and failed pages no longer break the layout.

- Sync reads every bookmark again. It had been stopping after the first 20
  and never reaching the end of the library.
- roost's icon is a drawer of index cards with a red tab on the saved one.
- A long or failed sync message appears in a notice under the toolbar
  instead of squeezing the search box. A failed sync says what went wrong
  in a sentence, with the original error under Technical details and Try
  again beside it.
- Long posts are cut to eight lines in the feed and Revisit, with Show
  more to read the rest.
- Posts that X had removed from your bookmarks are now called Missing from
  bookmarks.
- Images carry the descriptions their authors wrote on X, for VoiceOver.
  Posts you already kept gain them the first time this version opens.
- With a post open, Tab stays inside it, and closing it puts you back
  where you were.
- A setting that fails to save goes back to its old value and offers to
  try again.
- Duplicates stays readable when a post is filed in many collections.
- A link in brackets opens the page it points to.
- A collection with quotes in its name filters the author page correctly.
- Saved links that couldn't be fetched say why in a sentence.
- The media wall marks images that aren't on this Mac, and its Missing
  label no longer covers the video badge.
- Authors whose name starts with an emoji, or who have no name, get the
  right initial.
- Settings counts large media libraries in TB, and "1 file" in the
  singular.
- Revisit with nothing to show points you to Sync.
- Collection names are limited to 80 characters, field notes to 2,000 and
  highlights to 10,000. Longer names and notes you already have stay as
  they are.

## 0.2.0

Posts in other languages now read in English, search completes what you
type, and you can file a handful of posts at once.

- Posts written in another language show X's translation, with Show
  original a click away. Search finds them by the translated words too.
- Search suggests operators, authors and collections as you type; Tab or
  Enter takes one.
- Search reaches your notes and highlights, and three new operators narrow
  it: `domain:github.com`, `lang:ja` and `likes:1000`.
- All authors lists everyone you have kept posts from, and each author's
  page shows what you kept of theirs.
- Press `x` to select posts in the feed, then file them into a collection
  together.
- Share a collection as one Markdown file. Your private notes stay out of
  it unless you tick the box to include them.
- Edit, recolour and delete a collection from its heading.
- Highlights are marked in the post's own text, and each can carry a note.
- Suggestions for where a post belongs weigh what it means as well as its
  words, using Apple's language model on your Mac. They work on posts in
  English.
- With background sync on, roost reads every bookmark once a week to notice
  the ones gone from X. Read every bookmark in Settings does it now; it
  replaces the Full button in the toolbar.
- Press `?` to see every keyboard shortcut, and ⌘, to open Settings.
- A failed sync stays on screen with Try again, and a removed highlight can
  be brought back with Undo.
- The sidebar shows macOS's sidebar material, unless Reduce Transparency is
  on.
- Text is easier to read: secondary text and the accent meet 4.5:1
  contrast, no text is smaller than 11px, and buttons are set in the serif.
- Authors without a picture get a disc with their initial.
- Opening New collection and then changing your mind no longer leaves the
  field grabbing focus each time you move around.

## 0.1.0

The first version for friends.

- Sync your X bookmarks, in full or just the new ones, and keep them after
  they disappear from X.
- Organise them into collections, by hand or with saved searches, and sort
  the unsorted ones a key at a time with suggestions for where each belongs.
- Search with operators such as `has:video`, `is:unopened` and `from:`.
- Keep threads, the text of linked pages and every image, so posts still
  read right offline.
- Highlight passages, revisit old bookmarks, browse a media wall and find
  duplicates.
- Export everything as Markdown notes for Obsidian.
- Sync in the background from the menu bar and get a notification when new
  bookmarks arrive.
