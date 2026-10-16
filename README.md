# roost

A field journal for the posts you bookmark on X. roost keeps a copy of your
bookmarks on your Mac, so you can sort them into collections, search them,
read their threads and linked pages, and still have them after they vanish
from X. It needs no X Premium and no API key: it reads your bookmarks
through X's own website while you're signed in.

## Install

1. Download the latest `roost … .dmg` from
   [Releases](https://github.com/Ernxst/roost/releases/latest), open it and
   drag roost to Applications.
2. Open roost. macOS says it can't check roost for malicious software,
   because it isn't signed with a paid Apple developer account. Click
   **Done**.
3. Open **System Settings → Privacy & Security**, scroll to the message
   about roost, click **Open Anyway** and confirm.

You only do this once. After that roost updates itself: it checks once a
day, and **Check for Updates…** in its menu bar icon checks now.

On first sync roost shows X's sign-in page. Your session and library stay
on your Mac: roost talks only to X, to X's image servers and to the pages
your bookmarks link to.

## Develop

roost is a [mygo](https://github.com/egoist/mygo) app: Go for the library,
sync and storage, and Preact in the system webview for the interface.

```sh
bun install
bun run dev     # the development app, rebuilt as files change
go test ./...
bun run build   # a universal app and disk image in build/
```

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
