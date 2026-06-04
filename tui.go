// A small, dependency-free terminal menu. It uses raw (cbreak) mode via `stty`
// for single-key navigation and ANSI escapes for rendering — no tview/tcell, so
// the binary stays lean and cgo-free. When an action runs (capture, markup,
// record, upload), the menu drops back to normal cooked mode so the existing
// prompts and progress output work unchanged, then redraws.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// author is shown in the TUI header.
const author = "Prasul S"

type menuItem struct {
	key   string
	label string
}

func runTUI(cfg Config) error {
	if !isTTY(os.Stdin) {
		return fmt.Errorf("the menu needs an interactive terminal")
	}

	items := []menuItem{
		{"1", "Screenshot (region) → upload"},
		{"2", "Screenshot (region) → markup → upload"},
		{"3", "Window screenshot → upload"},
		{"4", "Full screen → upload"},
		{"5", "Record screen (ffmpeg) → upload"},
		{"6", "Upload a file (enter path)"},
		{"7", "Change destination"},
		{"q", "Quit"},
	}

	if err := enterRaw(); err != nil {
		return fmt.Errorf("terminal setup: %w", err)
	}
	defer leaveRaw()

	dest := cfg.DefaultDest
	sel := 0
	for {
		renderMenu(items, sel, dest)

		var chosen string
		switch readKey() {
		case "up":
			sel = clampIndex(sel-1, len(items))
		case "down":
			sel = clampIndex(sel+1, len(items))
		case "enter":
			chosen = items[sel].key
		case "quit":
			clearScreen()
			return nil
		default:
			if k := lastKey; k >= "1" && k <= "9" {
				if i := indexOfKey(items, k); i >= 0 {
					sel, chosen = i, k
				}
			}
		}
		if chosen == "" {
			continue
		}
		if chosen == "q" {
			clearScreen()
			return nil
		}

		// Run the action in normal cooked mode so prompts/progress work.
		leaveRaw()
		clearScreen()
		if nd := tuiAction(cfg, chosen, dest); nd != "" {
			dest = nd
		}
		fmt.Print(blue("\nPress Enter to return to the menu..."))
		bufio.NewReader(os.Stdin).ReadString('\n')
		enterRaw()
	}
}

// tuiAction performs the chosen menu action. It returns a non-empty string only
// when the destination was changed.
func tuiAction(cfg Config, key, dest string) string {
	switch key {
	case "1":
		captureThenUpload(cfg, dest, "interactive", false)
	case "2":
		captureThenUpload(cfg, dest, "interactive", true)
	case "3":
		captureThenUpload(cfg, dest, "window", false)
	case "4":
		captureThenUpload(cfg, dest, "full", false)
	case "5":
		secs := promptInt("Duration in seconds (0 = until you press q in ffmpeg): ")
		src, err := recordVideo(cfg.Video, secs)
		if err != nil {
			fmt.Println(red("record: " + err.Error()))
			return ""
		}
		tuiUpload(cfg, dest, src, false, true)
	case "6":
		p := expandHome(strings.TrimSpace(promptLine("File path to upload: ")))
		if p == "" {
			return ""
		}
		if _, err := os.Stat(p); err != nil {
			fmt.Println(red("no such file: " + p))
			return ""
		}
		tuiUpload(cfg, dest, p, false, false) // never delete a user-supplied file
	case "7":
		return chooseDest(cfg, dest)
	}
	return ""
}

func captureThenUpload(cfg Config, dest, mode string, markup bool) {
	src, err := capture(mode)
	if err != nil {
		fmt.Println(red("capture: " + err.Error()))
		return
	}
	if src == "" {
		fmt.Println(yellow("Capture cancelled."))
		return
	}
	tuiUpload(cfg, dest, src, markup, true)
}

func tuiUpload(cfg Config, dest, srcPath string, markup, cleanupSrc bool) {
	d, ok := cfg.Destinations[dest]
	if !ok {
		fmt.Println(red("no destination " + dest))
		return
	}
	up, err := uploaderFor(d)
	if err != nil {
		fmt.Println(red("destination: " + err.Error()))
		return
	}
	url, err := processAndUpload(cfg, up, srcPath, jobOpts{
		annotate:   markup,
		optimize:   cfg.Optimize,
		cleanupSrc: cleanupSrc,
	})
	if err != nil {
		fmt.Println(red("upload failed: " + err.Error()))
		return
	}
	presentResult(url, cfg.CopyToClipboard)
}

func chooseDest(cfg Config, current string) string {
	names := sortedDestNames(cfg)
	fmt.Println(bold("Destinations:"))
	for i, n := range names {
		marker := "  "
		if n == current {
			marker = green("* ")
		}
		fmt.Printf("  %d) %s%s (%s)\n", i+1, marker, n, cfg.Destinations[n].Type)
	}
	s := strings.TrimSpace(promptLine("Pick a number (Enter to keep current): "))
	if s == "" {
		return ""
	}
	if i, err := strconv.Atoi(s); err == nil && i >= 1 && i <= len(names) {
		fmt.Println(green("destination → " + names[i-1]))
		return names[i-1]
	}
	fmt.Println(yellow("invalid choice"))
	return ""
}

func sortedDestNames(cfg Config) []string {
	names := make([]string, 0, len(cfg.Destinations))
	for n := range cfg.Destinations {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ---------------------------------------------------------------------------
// Rendering + input (ANSI + stty raw mode)
// ---------------------------------------------------------------------------

func clearScreen() { fmt.Print("\033[2J\033[H") }

// --- soft modern (Catppuccin-ish) truecolor palette ---
func tc(r, g, b int, s string) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, s)
}
func clrTitle(s string) string   { return "\x1b[1m" + tc(137, 180, 250, s) } // blue, bold
func clrTagline(s string) string { return tc(186, 194, 222, s) }             // subtext
func clrVer(s string) string     { return tc(147, 153, 178, s) }             // overlay
func clrAuthor(s string) string  { return tc(166, 227, 161, s) }             // green
func clrClaude(s string) string  { return tc(250, 179, 135, s) }             // peach
func clrAccent(s string) string  { return tc(203, 166, 247, s) }             // mauve
func clrText(s string) string    { return tc(205, 214, 244, s) }             // text
func clrBorder(s string) string  { return tc(88, 91, 112, s) }               // surface
func clrHint(s string) string    { return tc(108, 112, 134, s) }             // overlay0

// headerRow lays out a left and right segment inside a bordered box row,
// padding from the segments' plain (uncolored) widths so alignment is correct.
func headerRow(leftPlain, rightPlain, leftColored, rightColored string, inner int) string {
	pad := inner - utf8.RuneCountInString(leftPlain) - utf8.RuneCountInString(rightPlain)
	if pad < 1 {
		pad = 1
	}
	return clrBorder("│ ") + leftColored + strings.Repeat(" ", pad) + rightColored + clrBorder(" │")
}

func renderHeader() {
	const inner = 56
	ver := version
	if ver != "Author" && !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}

	bar := clrBorder("─")
	top := clrBorder("┌") + strings.Repeat(bar, inner+2) + clrBorder("┐")
	bot := clrBorder("└") + strings.Repeat(bar, inner+2) + clrBorder("┘")
	claude := "✦ GoLang"

	fmt.Print("\r\n  " + top + "\r\n")
	fmt.Print("  " + headerRow("Scrnshot", author,
		clrTitle("Scrnshot"), clrAuthor(author), inner) + "\r\n")
	fmt.Print("  " + headerRow("A no nonsense screenshot program", claude,
		clrTagline("A no nonsense screenshot program"),
		clrAccent("✦ ")+clrClaude("Made with Golang"), inner) + "\r\n")
	fmt.Print("  " + headerRow(ver, "", clrVer(ver), "", inner) + "\r\n")
	fmt.Print("  " + bot + "\r\n")
}

func renderMenu(items []menuItem, sel int, dest string) {
	clearScreen()
	renderHeader()
	fmt.Print("\r\n  " + clrHint("destination: ") + clrAccent(dest) + "\r\n\r\n")
	for i, it := range items {
		if i == sel {
			fmt.Print(clrAccent("  ▌ ") + clrAccent(it.key) + "  " + clrAccent(it.label) + "\r\n")
		} else {
			fmt.Print("    " + clrVer(it.key) + "  " + clrText(it.label) + "\r\n")
		}
	}
	fmt.Print("\r\n  " + clrHint("↑/↓ or j/k · Enter or number to select · q to quit") + "\r\n")
}

func clampIndex(i, n int) int {
	if n == 0 {
		return 0
	}
	if i < 0 {
		return n - 1
	}
	if i >= n {
		return 0
	}
	return i
}

func indexOfKey(items []menuItem, key string) int {
	for i, it := range items {
		if it.key == key {
			return i
		}
	}
	return -1
}

// lastKey holds the most recent digit/char from readKey so the menu loop can
// treat number presses as direct selections.
var lastKey string

// readKey returns "up", "down", "enter", "quit", or "" — and sets lastKey to a
// digit string ("1".."9") when a number is pressed.
func readKey() string {
	lastKey = ""
	buf := make([]byte, 1)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return ""
	}
	b := buf[0]
	switch b {
	case '\r', '\n':
		return "enter"
	case 'k', 'K':
		return "up"
	case 'j', 'J':
		return "down"
	case 'q', 'Q':
		return "quit"
	case 0x1b: // ESC — possibly an arrow sequence (ESC [ A/B)
		seq := make([]byte, 2)
		if _, e := os.Stdin.Read(seq); e == nil && seq[0] == '[' {
			switch seq[1] {
			case 'A':
				return "up"
			case 'B':
				return "down"
			}
		}
		return ""
	}
	if b >= '1' && b <= '9' {
		lastKey = string(b)
	}
	return ""
}

func promptLine(label string) string {
	fmt.Print(label)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(s, "\r\n")
}

func promptInt(label string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(promptLine(label)))
	return n
}

// terminal raw-mode helpers via stty (no external Go deps).
var savedTTY string

func enterRaw() error {
	if out, err := sttyCapture("-g"); err == nil {
		savedTTY = strings.TrimSpace(out)
	}
	return stty("cbreak", "-echo")
}

func leaveRaw() {
	if savedTTY != "" {
		_ = stty(savedTTY)
		return
	}
	_ = stty("sane")
}

func stty(args ...string) error {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func sttyCapture(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return string(out), err
}
