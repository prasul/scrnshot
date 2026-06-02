// scrnshot — a ShareX-style capture/optimize/upload tool for macOS.
//
// It captures (via the built-in `screencapture`), optionally optimizes with
// ImageMagick, uploads through a pluggable destination (FTP/FTPS/SFTP/S3/HTTP),
// and copies the resulting share URL to the clipboard.
//
// Universal binary: pure Go core (the uploaders need no cgo); build with
// GOARCH=amd64 and arm64 and stitch with `lipo`, or let GoReleaser do it.
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/prasul/scrnshot/internal/uploader"
)

<<<<<<< HEAD
// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

=======
>>>>>>> f57a46f15431f22f04c5e46cbf02ca9868bc24ed
// ---------------------------------------------------------------------------
// Config — TWEAK ZONE lives in ~/.config/scrnshot/config.json
// ---------------------------------------------------------------------------

type Config struct {
	// Destinations are named; -dest picks one, otherwise DefaultDest is used.
	Destinations map[string]uploader.Destination `json:"destinations"`
	DefaultDest  string                          `json:"default_destination"`

	// Optimization (applied to png/jpg/jpeg).
	Optimize       bool `json:"optimize"`
	ResizePercent  int  `json:"resize_percent"`  // 0 or 100 = no resize
	Colors         int  `json:"colors"`          // 0 = full colour; 256 = palette
	PNGCompression int  `json:"png_compression"` // 0-9
	JPEGQuality    int  `json:"jpeg_quality"`    // 1-100

	CopyToClipboard bool `json:"copy_to_clipboard"`
}

func defaultConfig() Config {
	verifyFalse := false
	return Config{
		DefaultDest: "bigscoots",
		Destinations: map[string]uploader.Destination{
			"bigscoots": {
				Type:       "ftps",
				Host:       "38.58.227.202",
				User:       "PUT_NEW_USERNAME_HERE",
				Pass:       "PUT_NEW_PASSWORD_HERE",
				RemoteDir:  "/",
				VerifyCert: &verifyFalse,
				ShareURL:   "https://share.bigscoots.com/PUT_NEW_TOKEN_HERE",
			},
			"r2-example": {
				Type:      "s3",
				Endpoint:  "https://<accountid>.r2.cloudflarestorage.com",
				Region:    "auto",
				Bucket:    "screenshots",
				AccessKey: "...",
				SecretKey: "...",
				KeyPrefix: "shots/",
				ShareURL:  "https://cdn.example.com",
			},
			"http-example": {
				Type:       "http",
				URL:        "https://example.com/upload",
				FileField:  "file",
				Headers:    map[string]string{"Authorization": "Bearer TOKEN"},
				URLJSONKey: "data.url",
			},
		},
		Optimize:        true,
		ResizePercent:   80,
		Colors:          256,
		PNGCompression:  9,
		JPEGQuality:     90,
		CopyToClipboard: true,
	}
}

// ---------------------------------------------------------------------------
// colour helpers (auto-off when not a TTY)
// ---------------------------------------------------------------------------

var useColor = func() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}()

func col(code, s string) string {
	if !useColor {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func green(s string) string  { return col("32", s) }
func yellow(s string) string { return col("33", s) }
func blue(s string) string   { return col("36", s) }
func red(s string) string    { return col("31", s) }
func bold(s string) string   { return col("1", s) }

// ---------------------------------------------------------------------------

func main() {
	var (
		fConfig  = flag.String("config", defaultConfigPath(), "config file path")
		fDest    = flag.String("dest", "", "destination name from config (overrides default)")
		fFile    = flag.String("file", "", "upload this existing file instead of capturing")
		fCapture = flag.String("capture", "interactive", "capture mode: interactive | window | full")
		fNoOpt   = flag.Bool("no-optimize", false, "skip optimization")
		fNoClip  = flag.Bool("no-clipboard", false, "do not copy URL to clipboard")
		fKeep    = flag.Bool("keep", false, "keep the local file after upload")
		fList    = flag.Bool("list", false, "list configured destinations and exit")
<<<<<<< HEAD
		fVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *fVersion {
		fmt.Println("scrnshot", version)
		return
	}

=======
	)
	flag.Parse()

>>>>>>> f57a46f15431f22f04c5e46cbf02ca9868bc24ed
	cfg, err := loadConfig(*fConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("config: ")+err.Error())
		os.Exit(1)
	}

	if *fList {
		fmt.Println(bold("Destinations:"))
		for name, d := range cfg.Destinations {
			marker := "  "
			if name == cfg.DefaultDest {
				marker = green("* ")
			}
			fmt.Printf("%s%s (%s)\n", marker, name, d.Type)
		}
		return
	}

	destName := cfg.DefaultDest
	if *fDest != "" {
		destName = *fDest
	}
	dest, ok := cfg.Destinations[destName]
	if !ok {
		fmt.Fprintf(os.Stderr, red("no destination %q in config (try -list)\n"), destName)
		os.Exit(1)
	}

	up, err := uploader.New(dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("destination: ")+err.Error())
		os.Exit(1)
	}

	// 1. Obtain a file: either the one passed in, or a fresh capture.
	var srcPath string
	var cleanupSrc bool
	if *fFile != "" {
		srcPath = *fFile
	} else {
		srcPath, err = capture(*fCapture)
		if err != nil {
			fmt.Fprintln(os.Stderr, red("capture: ")+err.Error())
			os.Exit(1)
		}
		if srcPath == "" {
			fmt.Println(yellow("Capture cancelled."))
			return
		}
		cleanupSrc = true
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	beforeSize, _ := fileSize(srcPath)

	// 2. Optimize into a temp file if applicable.
	workPath := srcPath
	if cfg.Optimize && !*fNoOpt && optimizable(ext) {
		tmp := filepath.Join(os.TempDir(), "scrnshot-opt-"+mustRandom()+ext)
		if err := optimize(srcPath, tmp, ext, cfg); err != nil {
			fmt.Println(yellow("optimize failed (" + err.Error() + ") — uploading original"))
		} else {
			workPath = tmp
			defer os.Remove(tmp)
			after, _ := fileSize(tmp)
			fmt.Printf("%s %s -> %s\n", green("optimized"), humanSize(beforeSize), humanSize(after))
		}
	}

	// 3. Random remote name.
	remoteName := mustRandom() + ext

	// 4. Upload.
	fmt.Printf("%s via %s ...\n", blue("uploading"), bold(up.Kind()))
	f, err := os.Open(workPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("open: ")+err.Error())
		os.Exit(1)
	}
	sz, _ := fileSize(workPath)
	shareURL, err := up.Upload(remoteName, f, sz)
	f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, red("upload failed: ")+err.Error())
		fmt.Fprintln(os.Stderr, yellow("local file kept at: ")+srcPath)
		os.Exit(1)
	}

	// 5. Output + clipboard.
	fmt.Println()
	fmt.Println(green("======================="))
	fmt.Println(shareURL)
	fmt.Println(green("======================="))

	if cfg.CopyToClipboard && !*fNoClip {
		if err := copyClipboard(shareURL); err != nil {
			fmt.Println(yellow("(clipboard skipped: " + err.Error() + ")"))
		} else {
			fmt.Println(blue("(URL copied to clipboard)"))
		}
	}

	if cleanupSrc && !*fKeep {
		_ = os.Remove(srcPath)
	}
}

// ---------------------------------------------------------------------------
// Capture via macOS built-in `screencapture`
// ---------------------------------------------------------------------------

func capture(mode string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("capture mode needs macOS; use -file on other platforms")
	}
	out := filepath.Join(os.TempDir(), "scrnshot-cap-"+mustRandom()+".png")

	args := []string{}
	switch mode {
	case "interactive":
		args = append(args, "-i") // region or window selector, like Cmd-Shift-4
	case "window":
		args = append(args, "-iW")
	case "full":
		// no flag = whole screen
	default:
		return "", fmt.Errorf("unknown capture mode %q", mode)
	}
	args = append(args, out)

	cmd := exec.Command("screencapture", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	// If the user pressed Esc, screencapture exits 0 but writes no file.
	if _, err := os.Stat(out); err != nil {
		return "", nil
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Optimization (single ImageMagick pass)
// ---------------------------------------------------------------------------

func optimizable(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

func optimize(src, dst, ext string, cfg Config) error {
	args := []string{src, "-strip", "-interlace", "Plane"}
	if cfg.ResizePercent > 0 && cfg.ResizePercent != 100 {
		args = append(args, "-resize", fmt.Sprintf("%d%%", cfg.ResizePercent))
	}
	switch ext {
	case ".png":
		if cfg.Colors > 0 {
			args = append(args, "-colors", fmt.Sprintf("%d", cfg.Colors))
		}
		args = append(args,
			"-define", "png:compression-filter=5",
			"-define", fmt.Sprintf("png:compression-level=%d", cfg.PNGCompression),
			"-define", "png:compression-strategy=1",
		)
	case ".jpg", ".jpeg":
		args = append(args, "-quality", fmt.Sprintf("%d", cfg.JPEGQuality))
	}
	args = append(args, dst)

	cmd := exec.Command("magick", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Config IO
// ---------------------------------------------------------------------------

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "scrnshot", "config.json")
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if werr := writeTemplate(path); werr != nil {
			return Config{}, werr
		}
		return Config{}, fmt.Errorf("no config — wrote a template to %s; fill it in and run again", path)
	}
	if err != nil {
		return Config{}, err
	}
	cfg := defaultConfig()
	cfg.Destinations = nil // let the file fully define destinations
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	if len(cfg.Destinations) == 0 {
		return Config{}, fmt.Errorf("config has no destinations")
	}
	return cfg, nil
}

func writeTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(defaultConfig(), "", "  ")
	return os.WriteFile(path, data, 0o600) // holds credentials
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mustRandom() string {
	b := make([]byte, 14)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func copyClipboard(s string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("pbcopy is macOS-only")
	}
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader([]byte(s))
	return cmd.Run()
}

func fileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGT"[exp])
}
