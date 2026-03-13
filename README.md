# cites

Citation reference tooling for research documents. Captures source documents (HTML, PDF, TXT), extracts plain text with metadata, and renders HTML reference pages with line-numbered, anchor-linked text.

## Usage

```bash
# Capture a source document
cites capture https://example.com/document.pdf "Document Title"
cites capture local-file.html "Local Document" output.txt

# Render to HTML reference page
cites render research/sources/document.txt docs/document.html

# Check if source has changed upstream
cites check research/sources/document.txt
cites check --update research/sources/document.txt  # also update body
```

## Features

- **Three source types**: HTML, PDF (via `pdftotext`), plain text
- **Section detection**: automatic heading detection with original anchor mapping
- **PDF page boundaries**: page-level metadata with side-by-side render layout
- **Content hashing**: SHA-256 hash for change detection
- **Version tracking**: history of content changes over time
- **Line anchors**: every line addressable via `#L42` URLs

## Structure

```
cmd/cites/          ← CLI entry point
internal/
  source/           ← source file format (YAML frontmatter + body)
  capture/          ← text extraction and heading detection
  render/           ← HTML reference page generation
research/
  demo.md           ← APR vs AER comparison (demo article)
  sources/          ← captured source files
docs/               ← generated HTML (gitignored)
```

## Build

```bash
task build          # build CLI to bin/cites
task test           # run unit tests
task check          # fmt + vet + test
task docs:build     # render all sources to HTML
```

Requires `poppler-utils` for PDF extraction (`pdftotext`).

## Links

- Source: https://codeberg.org/hum3/cites
- Mirror: https://github.com/drummonds/cites
