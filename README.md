# scrnshot
<<<<<<< HEAD
=======
<<<<<<< HEAD
=======
<<<<<<< HEAD
>>>>>>> f57a46f15431f22f04c5e46cbf02ca9868bc24ed
>>>>>>> 2e8b1664db4249d004dea793b3e13c8d8f22bd19

A ShareX-style capture → optimize → upload tool for macOS. Capture a region,
optimize the image, upload it to your own server, and get the share URL on your
clipboard — in one keypress. Ships as a **universal binary** that runs natively
on both Intel and Apple Silicon Macs.

ShareX is Windows-only; this fills the same niche on the Mac with an open,
scriptable uploader model.

## What it does

1. Captures via the built-in macOS `screencapture` (interactive region, window,
   or full screen) — no extra capture app needed.
2. Optionally optimizes PNG/JPEG with ImageMagick (resize, palette reduction,
   compression) to shrink share-link payloads.
3. Renames to an unguessable random name and uploads through a pluggable
   destination.
4. Copies the resulting URL to the clipboard.

## Pluggable destinations (ShareX-style)

A single config file describes one or more named destinations. Supported types:

| type   | use it for                                        |
|--------|---------------------------------------------------|
| `ftp`  | plain FTP                                         |
| `ftps` | FTP over explicit TLS (e.g. BigScoots share)      |
| `sftp` | SSH file transfer (password or private key)       |
| `s3`   | AWS S3 and S3-compatible: Cloudflare R2, Backblaze B2, Wasabi, MinIO |
| `http` | any custom HTTP POST endpoint, with the URL pulled from a JSON key or regex |

The S3 backend implements AWS Signature V4 with the standard library only — no
AWS SDK, so the binary stays small.

## Install

Requires [ImageMagick](https://imagemagick.org) only if you want optimization
(`brew install imagemagick`).

### From a release

Download the universal tarball from the Releases page, then:

```sh
tar xzf scrnshot_*_macos_universal.tar.gz
install -m 0755 scrnshot ~/bin/scrnshot   # any dir on your PATH
```

### From source

```sh
git clone https://github.com/prasul/scrnshot && cd scrnshot
make universal     # builds the Intel+ARM universal binary with lipo
make install       # copies to ~/bin
```

## Configure

Run it once to generate a template:

```sh
scrnshot          # writes ~/.config/scrnshot/config.json (mode 0600) and exits
```

Edit that file (see `config.example.json` for all destination types), then:

```sh
scrnshot                 # capture interactively, upload to the default dest
scrnshot -dest my-sftp   # use a specific destination
scrnshot -capture window # capture a window instead of a region
scrnshot -file shot.png  # upload an existing file (no capture)
scrnshot -list           # show configured destinations
scrnshot -no-optimize    # upload as captured
```

## Bind it to a hotkey

`scrnshot` is a plain CLI, so any launcher can trigger it system-wide:

- **macOS Shortcuts**: new shortcut → "Run Shell Script" → `~/bin/scrnshot` →
  assign a keyboard shortcut.
- **Raycast / Alfred**: add a script command pointing at the binary.

This avoids the accessibility/hotkey permission prompt a native global hotkey
would require. (A built-in hotkey is on the roadmap.)

## Record the screen

`scrnshot` can also record video, optimize it, and upload it the same way.
Recording uses [ffmpeg](https://ffmpeg.org) (`brew install ffmpeg`).

```sh
scrnshot -list-screens        # find your "Capture screen N" index
scrnshot -record              # record until you press Enter (or Ctrl-C)
scrnshot -record -duration 15 # record a fixed 15 seconds
```

The encoder settings in the config's `video` block *are* the optimization:
`h264_videotoolbox` hardware-encodes with low CPU (use `hevc_videotoolbox` for
smaller files), `faststart` lets the uploaded clip play before it finishes
downloading, and `scale_percent` can shrink the resolution. The screen index
comes from `-list-screens`; set it once in `screen_index`.

Recording needs the same Screen Recording permission as screenshots — granted
to the terminal (or launcher) that runs `scrnshot`.

## Roadmap

- `-watch` mode: auto-process new screenshots dropped in a folder.
- Menu-bar app (systray) with recent-uploads and copy-URL.
- Native global hotkey + region overlay.

## Notes

- The config file holds credentials; it is written `0600`. Keep it out of
  version control.
- `verify_cert: false` on FTPS/S3/HTTP disables TLS certificate verification —
  it mirrors the original lftp `verify-certificate no`. Prefer leaving it
  unset (verification on) unless your server needs it off.
<<<<<<< HEAD
=======
<<<<<<< HEAD
=======
=======
A terminal level screenshot tool that helps you upload the files on to remote SFTP server
>>>>>>> 1e94d5490cf54d9450ebf86b0725dccec815daeb
>>>>>>> f57a46f15431f22f04c5e46cbf02ca9868bc24ed
>>>>>>> 2e8b1664db4249d004dea793b3e13c8d8f22bd19
