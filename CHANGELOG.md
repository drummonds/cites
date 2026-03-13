# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Added

- Initial project structure with three-package architecture (source, capture, render)
- CLI with `capture`, `render`, and `check` commands
- Source file format: YAML frontmatter + plain text body
- Text extraction from HTML (via golang.org/x/net), PDF (via pdftotext), and plain text
- Section heading detection with regex-based pattern matching
- PDF page boundary detection via form feed characters
- `pages` metadata field for PDF page-to-line mapping
- Side-by-side HTML render layout for PDF sources with page navigation
- Page navigation links to source PDF at specific pages (`url#page=N`)
- SHA-256 content hashing with change detection
- Check command: re-fetch from URL, compare hash, track versions
- `--update` flag on check: replace body and hash when source changes
- Unit tests for source (parse/marshal round-trip, hash, sections, pages)
- Unit tests for capture (HTML extraction, section detection, slugify, page boundaries)
- Unit tests for render (HTML output, side-by-side layout, line anchors, page breaks)
- Test fixtures: sample HTML, TXT, and complete source file with frontmatter
- `docs:build` task renders all captured sources to HTML
- Demo research article: APR vs AER comparison with cross-source citations
- Exported `Extract` function from capture package for use by check command
