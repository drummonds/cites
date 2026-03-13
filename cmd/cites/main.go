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
  cites render  <source-file> [output-path]
  cites check   [--update] <source-file>`)
}

func cmdCapture(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cites capture <url-or-file> <title> [output-path]")
		os.Exit(1)
	}

	location := args[0]
	title := args[1]

	src, err := capture.Capture(location, title)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture failed: %v\n", err)
		os.Exit(1)
	}

	outPath := ""
	if len(args) >= 3 {
		outPath = args[2]
	} else {
		// Default: slugify title + .txt
		slug := strings.ToLower(title)
		slug = strings.ReplaceAll(slug, " ", "-")
		outPath = slug + ".txt"
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
		fmt.Fprintln(os.Stderr, "usage: cites render <source-file> [output-path]")
		os.Exit(1)
	}

	srcPath := args[0]
	src, err := source.ParseFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	outPath := ""
	if len(args) >= 2 {
		outPath = args[1]
	} else {
		base := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		outPath = base + ".html"
	}

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating %s: %v\n", outPath, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := render.Render(f, src); err != nil {
		fmt.Fprintf(os.Stderr, "rendering: %v\n", err)
		os.Exit(1)
	}

	status := "ok"
	if src.HasChanged() {
		status = "CHANGED"
	}
	fmt.Printf("rendered %s → %s (hash: %s)\n", srcPath, outPath, status)
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
