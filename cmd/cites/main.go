package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codeberg.org/hum3/cites/internal/capture"
	"codeberg.org/hum3/cites/internal/render"
	"codeberg.org/hum3/cites/internal/source"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "capture":
		cmdCapture(os.Args[2:])
	case "render":
		cmdRender(os.Args[2:])
	case "check":
		cmdCheck(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `cites — citation reference tooling

Usage:
  cites capture <url-or-file> <title> [output-path]
  cites render  [--base PATH] [--pdf PATH] [--lines-per-page N] <source.yaml> <output-dir>
  cites check   [--update] <source-file>`)
}

func cmdCapture(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cites capture <url-or-file> <title> [output-path]")
		os.Exit(1)
	}

	location := args[0]
	title := args[1]

	outPath := ""
	if len(args) >= 3 {
		outPath = args[2]
	} else {
		// Default: slugify title + .yaml
		slug := strings.ToLower(title)
		slug = strings.ReplaceAll(slug, " ", "-")
		outPath = slug + ".yaml"
	}

	// Derive PDF save path from output path.
	var opts capture.CaptureOpts
	outDir := filepath.Dir(outPath)
	pdfDir := filepath.Join(outDir, "pdfs")
	base := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
	opts.PdfSavePath = filepath.Join(pdfDir, base+".pdf")

	src, err := capture.Capture(location, title, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture failed: %v\n", err)
		os.Exit(1)
	}

	// Make OriginalFile relative to the source file location.
	if src.Meta.OriginalFile != "" {
		rel, relErr := filepath.Rel(outDir, src.Meta.OriginalFile)
		if relErr == nil {
			src.Meta.OriginalFile = rel
		}
	}

	if err := src.WriteFile(outPath); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("captured %s → %s (%d lines, %d sections)\n",
		location, outPath, len(strings.Split(src.Body, "\n")), len(src.Meta.Sections))
}

func cmdRender(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cites render [--base PATH] [--pdf PATH] [--lines-per-page N] <source.yaml> <output-dir>")
		os.Exit(1)
	}

	opts := render.RenderOpts{}
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--base":
			if i+1 < len(args) {
				opts.BasePath = args[i+1]
				i++
			}
		case "--pdf":
			if i+1 < len(args) {
				opts.PdfPath = args[i+1]
				i++
			}
		case "--lines-per-page":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.LinesPerPage)
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cites render [--base PATH] [--pdf PATH] [--lines-per-page N] <source.yaml> <output-dir>")
		os.Exit(1)
	}

	srcPath := positional[0]
	outDir := positional[1]

	src, err := source.ParseFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	// Default --pdf: resolve original_file from frontmatter relative to source file location.
	if opts.PdfPath == "" && src.Meta.OriginalFile != "" {
		opts.PdfPath = filepath.Join(filepath.Dir(srcPath), src.Meta.OriginalFile)
	}

	if err := render.RenderDir(outDir, src, opts); err != nil {
		fmt.Fprintf(os.Stderr, "rendering: %v\n", err)
		os.Exit(1)
	}

	status := "ok"
	if src.HasChanged() {
		status = "CHANGED"
	}
	pages := render.PagesForSource(src, opts.LinesPerPage)
	fmt.Printf("rendered %s → %s/ (%d pages, hash: %s)\n", srcPath, outDir, len(pages), status)
}

func cmdCheck(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cites check [--update] <source-file>")
		os.Exit(1)
	}

	update := false
	var srcPath string
	for _, a := range args {
		if a == "--update" {
			update = true
		} else {
			srcPath = a
		}
	}
	if srcPath == "" {
		fmt.Fprintln(os.Stderr, "usage: cites check [--update] <source-file>")
		os.Exit(1)
	}

	src, err := source.ParseFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	// If no URL, can only do local hash check.
	if src.Meta.URL == "" {
		if src.HasChanged() {
			fmt.Printf("%s: CHANGED (stored hash does not match body)\n", srcPath)
			os.Exit(1)
		}
		fmt.Printf("%s: ok (local only, no URL to re-fetch)\n", srcPath)
		return
	}

	// Re-fetch from source URL and extract text.
	_, newBody, err := capture.Extract(src.Meta.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "re-fetching %s: %v\n", src.Meta.URL, err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	src.Meta.LastChecked = now
	newHash := source.ContentHash(newBody)
	changed := newHash != src.Meta.ContentHash

	if changed {
		fmt.Printf("%s: CHANGED (upstream content differs)\n", srcPath)
		src.Meta.Versions = append(src.Meta.Versions, source.Version{
			Date: now.Format("2006-01-02"),
			Hash: newHash,
			Note: "Content changed on re-check",
		})
		if update {
			src.Body = newBody
			src.Meta.ContentHash = newHash
			fmt.Printf("  updated body and hash\n")
		}
	} else {
		fmt.Printf("%s: ok\n", srcPath)
	}

	if err := src.WriteFile(srcPath); err != nil {
		fmt.Fprintf(os.Stderr, "writing %s: %v\n", srcPath, err)
		os.Exit(1)
	}
}
