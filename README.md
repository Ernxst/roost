# roost

roost keeps the posts you bookmark on X, on your Mac, for good. It copies
your bookmarks into a library you can sort into collections, search, and
read at your own pace, with their threads, pictures and linked articles.
Posts stay in roost even after their authors delete them or X takes them
down.

You don't need to pay for X or set anything up. roost reads your bookmarks
the same way you do, by signing in to the X website.

![roost's feed: kept posts in three columns, with collections, rituals and the most-kept authors in the sidebar](design/screenshot.png)

## What you need

- A Mac running macOS 12 Monterey or later. Apple silicon and Intel Macs
  both work.
- An X account with some bookmarks.

## Install

1. Go to the [latest release](https://github.com/Ernxst/roost/releases/latest)
   and, under **Assets**, download the file whose name ends in `.dmg`.
2. Open the downloaded file. In the window that appears, drag the roost
   icon onto the Applications folder.
3. Open roost from your Applications folder. macOS warns that it can't
   check roost for malicious software, because roost isn't registered
   with Apple, which costs a yearly fee. Click **Done**.
4. Open **System Settings**, choose **Privacy & Security**, and scroll down
   to the message about roost. Click **Open Anyway**, then confirm with your
   password or Touch ID.

You only do this once. From then on roost opens like any other app and
updates itself.

## Your first sync

1. Click **Sync from X**.
2. A window shows X's sign-in page. Sign in as you normally would. roost
   remembers the sign-in, so you won't be asked again unless X signs you
   out.
3. roost reads your bookmarks, newest first. A large collection of
   bookmarks takes a few minutes.

After that, click **Sync** in the top corner whenever you want the latest
bookmarks. To have roost check on its own, open **Settings** and choose how
often under **Background sync**. roost keeps working from its icon in the
menu bar when you close its window, and can send you a notification when
new bookmarks arrive.

## Finding your way around

- **Collections** are folders for your bookmarks. Click **+ New
  collection** in the sidebar, then file any post into it from the post
  itself.
- **Sort the unsorted** shows your unfiled bookmarks one at a time. Press a
  number key to file each one into a collection; roost suggests where each
  one probably belongs.
- **Search** finds words in posts, in the articles they link to and in your
  own notes. A few extra words narrow it down:

  | Type | To find |
  | --- | --- |
  | `from:someone` | posts by that account |
  | `has:video` | posts with a video (or `photo`, `link`, `thread`) |
  | `in:"Reading list"` | posts in that collection |
  | `is:unopened` | posts you haven't opened in roost yet |

- **Revisit** brings back posts from this day in past years and ones you
  never got round to.
- **Highlights** collects passages you've marked in posts and articles.
- **Media wall** shows every photo and video you've kept, and
  **Duplicates** finds bookmarks that share a post, a link or a picture.
- Open any post to read its whole thread, write yourself a note, or read a
  linked article without leaving roost.
- **Export** in Settings saves your whole library as notes you can open in
  Obsidian or any other notes app.
- Press `?` to see every keyboard shortcut.

## Updates

roost checks for a new version once a day. When there is one, a window
shows what's new and offers to restart roost with it.
To check now, click the roost icon in the menu bar and choose **Check for
Updates…**.

## Privacy

Your bookmarks, notes and sign-in stay on your Mac. roost connects only to
X, to the servers X keeps its pictures on, and to the pages your bookmarks
link to. Nothing is sent to anyone else.

## If something goes wrong

- **macOS won't open roost.** Follow step 4 of Install again; the **Open
  Anyway** button appears for about an hour after the warning.
- **A sync fails.** roost says why and offers **Try again**. If X has
  signed you out, it shows the sign-in page again.
- **Some old bookmarks are marked "Missing from bookmarks".** You
  unbookmarked them on X, or X removed them. roost keeps its copy.

## Remove roost

Drag roost from Applications to the Bin. Your library stays in
`~/Library/Application Support/roost`; delete that folder too if you want
to remove your saved bookmarks.

## Develop

roost is a [mygo](https://github.com/egoist/mygo) app: Go for the library,
sync and storage, and Preact in the system webview for the interface.

```sh
bun install
bun run dev     # the development app, rebuilt as files change
go test ./...
bun run build   # a universal app and disk image in build/
```

`bun run dev:web` serves the interface alone at http://localhost:5287. Add
`?data=sample` for a made-up library, the one in the screenshot above, or
`?data=worst`, `empty`, `one` or `thousand` to test long names, empty views
and large libraries. A switch at the foot of the page moves between them.

## Release

1. Raise `version` in `mygo.config.ts` and add its section to
   `CHANGELOG.md`; the update window shows it as the release notes.
2. Run `bun run build -- -upload`. It signs the update archives with the key
   in `~/Library/Application Support/mygo/update-keys` and uploads them with
   the disk image to a draft GitHub release.
3. Publish the draft. Installed copies pick it up within a day.

The key that signs updates is the only one installed copies trust: keep a
copy in a password manager, because losing it means friends have to
reinstall by hand.
