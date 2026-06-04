# scrnshot

A ShareX-style **capture → optimize → upload** tool for macOS. Take a screenshot
or hand it a screen recording, have it optimized, uploaded to your own server,
and the share URL copied to your clipboard. Ships as a **universal binary** that
runs natively on both Intel and Apple Silicon Macs.

ShareX is Windows-only; `scrnshot` fills the same niche on the Mac with an open,
scriptable uploader model and no required cloud account.

## What it does

- **Screenshots** via the built-in macOS `screencapture` (interactive region,
  window, or full screen) — no extra capture app needed.
- **Screen recordings** — record with macOS (Cmd-Shift-5), then optimize and
  upload the file. Transcoding uses the built-in `avconvert`, so **no ffmpeg is
  required**. (An optional ffmpeg-based recorder is also included.)
- **Image optimization** with ImageMagick (resize, palette reduction,
  compression) to shrink share-link payloads.
- **Pluggable uploaders**: FTP, FTPS, SFTP, S3-compatible, or any custom HTTP
  endpoint — chosen per named destination in one config file.
- Renames every file to an unguessable random name, uploads it, and copies the
  resulting URL to the clipboard.

## Pluggable destinations

A single config file describes one or more named destinations. Pick one with
`-dest`, or set a `default_destination`.

| type   | use it for                                                                  |
|--------|-----------------------------------------------------------------------------|
| `ftp`  | plain FTP                                                                   |
| `ftps` | FTP over explicit TLS (`AUTH TLS`)                                          |
| `sftp` | SSH file transfer (password or private key)                                |
| `s3`   | AWS S3 and S3-compatible: Cloudflare R2, Backblaze B2, Wasabi, MinIO        |
| `http` | any HTTP POST endpoint; the share URL is read from a JSON key or a regex   |

The S3 backend implements AWS Signature V4 using only the standard library — no
AWS SDK, so the binary stays small. See `config.example.json` for a fully
filled-in example of every type.

## Install

Optional external tools, installed only if you use the matching feature:

- [ImageMagick](https://imagemagick.org) (`brew install imagemagick`) — image optimization.
- `avconvert` — video optimization; **ships with macOS**, nothing to install.
- [ffmpeg](https://ffmpeg.org) (`brew install ffmpeg`) — only for the optional `-record` mode.

### Quick install (latest binary)

The latest release publishes a ready-to-run macOS universal binary at a stable
URL, so one line installs it:

```sh
curl -L https://github.com/prasul/scrnshot/releases/latest/download/scrnshot -o scrnshot
chmod +x scrnshot
mv scrnshot ~/bin/scrnshot      # any dir on your PATH
```

(Downloading via `curl` doesn't set macOS quarantine, so the binary runs without
a Gatekeeper prompt. It's universal — runs natively on Intel and Apple Silicon.)

### From the versioned archive

Each release also has `scrnshot_<version>_macos_universal.tar.gz`, which bundles
the README and example config:

```sh
tar xzf scrnshot_*_macos_universal.tar.gz
install -m 0755 scrnshot ~/bin/scrnshot
```

### From source

```sh
git clone https://github.com/prasul/scrnshot && cd scrnshot
make universal     # builds the Intel+ARM universal binary with lipo
make install       # copies to ~/bin
```

## Configure

Run it once to generate a config template, then fill it in:

```sh
scrnshot           # writes ~/.config/scrnshot/config.json (mode 0600) and exits
```

The file is written `0600` because it holds your upload credentials. Edit it
(see `config.example.json` for all options), then you're ready to go.

## Usage

```sh
scrnshot                       # capture a region, optimize, upload to default dest
scrnshot -dest my-sftp         # use a specific destination
scrnshot -capture window       # capture a window
scrnshot -capture full         # capture the whole screen
scrnshot -file shot.png        # upload an existing image (optimized)
scrnshot -file clip.mov        # optimize (avconvert) and upload a recording
scrnshot -no-optimize          # upload exactly as captured
scrnshot -list                 # list configured destinations
scrnshot -version              # print version
```

### Flags

| flag            | meaning                                                         |
|-----------------|-----------------------------------------------------------------|
| `-dest`         | destination name from the config (overrides `default_destination`) |
| `-tui`          | launch the interactive menu (also the default with no args in a terminal) |
| `-file`         | upload an existing file instead of capturing                    |
| `-capture`      | capture mode: `interactive` (default), `window`, `full`         |
| `-record`       | record a screen video via ffmpeg instead of a screenshot        |
| `-duration`     | recording length in seconds (`0` = until you press Enter)       |
| `-list-screens` | list ffmpeg/avfoundation capture devices                        |
| `-no-optimize`  | skip optimization, upload as-is                                 |
| `-annotate`     | open the capture in Preview for markup before upload            |
| `-no-clipboard` | don't copy the URL to the clipboard                             |
| `-keep`         | keep the local file after upload                                |
| `-list`         | list configured destinations                                    |
| `-config`       | use a non-default config path                                   |
| `-version`      | print version and exit                                          |

## Screen recordings

The recommended path needs no ffmpeg. Record with macOS's built-in recorder
(Cmd-Shift-5), then hand the file to scrnshot:

```sh
scrnshot -file ~/Desktop/recording.mov   # transcode with avconvert, then upload
```

The `video.preset` in your config controls the result:

- `Preset1920x1080` — 1080p H.264. Small and plays everywhere; ideal for links.
- `Preset1280x720` — smaller still.
- `PresetHEVCHighestQuality` — HEVC; smallest files, best in Safari.

avconvert never upscales, so a preset acts as a ceiling. List them all with
`avconvert --listPresets`. The output container comes from `video.container`
(`mp4` / `mov` / `m4v`).

### Optional: record directly (needs ffmpeg)

```sh
scrnshot -list-screens          # find your screen index
scrnshot -record -duration 15   # record, then optimize + upload
```

This uses ffmpeg + VideoToolbox. On 5K/Retina displays, set
`video.scale_percent` (e.g. `50`) so the width drops under the 4096px hardware
encoder limit; `-allow_sw 1` is passed automatically as a software fallback.

## Interactive menu

Run `scrnshot` with no arguments in a terminal and it opens a menu where you can
pick what to do — no flags to remember:

```
  scrnshot
  destination: bigscoots

  1  Screenshot (region) → upload
  2  Screenshot (region) → markup → upload
  3  Window screenshot → upload
  4  Full screen → upload
  5  Record screen (ffmpeg) → upload
  6  Upload a file (enter path)
  7  Change destination
  q  Quit
```

Navigate with ↑/↓ or `j`/`k`, select with Enter or the number key, quit with
`q`. After each action it shows the share URL and returns to the menu. Force it
with `-tui`. The menu is dependency-free (no tview/tcell) — it uses raw terminal
mode and ANSI, so the binary stays lean.

When launched from a hotkey or launcher (no terminal attached), `scrnshot` skips
the menu and runs a normal single capture, so flag-based usage still works for
Shortcuts/Raycast.

## Markup (arrows, highlights, text)

Add `-annotate` to open the capture in **Preview** before it uploads. Use
Preview's Markup toolbar (the pen-tip icon) to draw arrows, boxes, highlights,
text, or blur, then save with **Cmd-S** and press **Enter** in the terminal —
scrnshot picks up the edited image and uploads it.

```sh
scrnshot -annotate              # capture, mark up in Preview, then upload
scrnshot -file shot.png -annotate
```

Set `"annotate": true` in the config to make markup the default for every
capture. To use a different editor (e.g. Shottr, CleanShot, or `open -a`
another app), set `"editor"` in the config — the image path is appended to that
command. Markup applies to images only; it's skipped for recordings.

## Permissions

Both screenshots and recordings need the macOS **Screen Recording** permission,
and it attaches to *the app that launches the capture* — your terminal, not the
`scrnshot` binary itself. Grant it under System Settings → Privacy & Security →
**Screen & System Audio Recording**, enable your terminal (or the launcher you
bind the hotkey to), then fully quit and relaunch that app. Without it, captures
show only the desktop wallpaper. On Sequoia, expect a re-confirmation prompt
roughly once a month.

## Bind it to a hotkey

`scrnshot` is a plain CLI, so any launcher can trigger it system-wide:

- **macOS Shortcuts**: new shortcut → "Run Shell Script" → `~/bin/scrnshot` → assign a key.
- **Raycast / Alfred**: add a script command pointing at the binary.

Remember the permission above follows the launcher, so grant Screen Recording to
Shortcuts/Raycast if you trigger it that way.

## Releasing

Releases are tag-driven via GoReleaser (see `.goreleaser.yaml` and
`.github/workflows/release.yml`). Pushing a `v*` tag builds the macOS universal
binary, generates grouped release notes from `feat:`/`fix:` commits, and creates
a draft GitHub Release with the tarball and checksums.

```sh
git push origin main
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0          # triggers the release workflow
```

Run `go mod tidy` and commit `go.mod` + `go.sum` before tagging so the build has
a complete module graph.

## Notes

- The config file holds credentials and is written `0600`; keep it out of version control.
- `verify_cert: false` on FTPS/S3/HTTP disables TLS certificate verification.
  Leave it unset (verification on) unless your server genuinely needs it off.
- Some FTP servers reject `AUTH TLS`. If `type: ftps` fails with a 504 during
  connect, the server likely doesn't support explicit FTPS on that port — use
  `type: ftp` (plain), or ask your host about SFTP for an encrypted alternative.

## Roadmap

- `-watch` mode: auto-process new screenshots dropped in a folder.
- Menu-bar app (systray) with recent uploads and copy-URL.
- Native global hotkey + region overlay.
