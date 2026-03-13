# cites

Citation reference tooling: capture source documents and render HTML reference pages.

## Architecture

Two-phase pipeline with optional check:

### Phase 1: Capture (`cites capture <url-or-file> <title> [output-path]`)

- Input: URL or local file (HTML, PDF, TXT)
- Extracts plain text, detects headings/sections with original anchors
- For PDFs: detects page boundaries via form feed characters, records `pages` metadata
- Computes SHA-256 content hash
- Records metadata in YAML frontmatter
- Output: `.txt` file with frontmatter + extracted text in `research/sources/`

### Phase 2: Render (`cites render <source-file> [output-path]`)

- Input: captured source file
- Output: HTML reference document in `docs/`
  - Metadata header block (title, URL, dates, hash status)
  - Line-numbered text with per-line anchors (`#L42`)
  - Each line number links to closest section anchor in original source
  - Change detection: re-hash content, flag if source has been modified
  - **PDF sources with page data**: side-by-side layout with page navigation linking to source PDF pages

### Phase 3: Check (`cites check [--update] <source-file>`)

- Re-fetches source from URL, extracts text, compares hash
- Updates `last_checked` timestamp
- Records new version if content changed
- `--update` flag: also replaces body and content hash (non-destructive by default)
- Falls back to local hash check if no URL

## Source file format

```yaml
---
title: "AER Practice Note"
url: "https://www.ukfinance.org.uk/system/files/..."
source_type: pdf          # pdf | html | txt
source_date: "2024-01-15" # date of the source document itself
first_captured: "2025-03-13T10:30:00Z"
last_checked: "2025-03-13T10:30:00Z"
content_hash: "sha256:abc123..."
sections:
  - line: 1
    heading: "Title"
    original_anchor: ""
  - line: 15
    heading: "1. Introduction"
    original_anchor: "#page=3"
pages:
  - page: 1
    start_line: 1
  - page: 2
    start_line: 34
versions:
  - date: "2025-03-13"
    hash: "sha256:abc123..."
    note: "Initial capture"
---
[plain text content]
```

### Field reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | yes | Human-readable document title |
| `url` | string | yes | Canonical URL of original source |
| `source_type` | enum | yes | `pdf`, `html`, or `txt` |
| `source_date` | string | no | Date of the source document (YYYY-MM-DD) |
| `first_captured` | datetime | yes | When first captured (RFC 3339) |
| `last_checked` | datetime | yes | When last verified against source |
| `content_hash` | string | yes | `sha256:<hex>` of body text |
| `sections` | list | no | Detected headings with line numbers and original anchors |
| `pages` | list | no | PDF page boundaries (page number → start line) |
| `versions` | list | no | Version history (date, hash, note) |

## Directory layout

```
cmd/cites/          ← CLI (capture, render, check)
internal/
  source/           ← source file format (parse/serialize frontmatter + body)
  capture/          ← text extraction from HTML/PDF/TXT, heading detection, page boundaries
  render/           ← HTML reference document generation (standard + side-by-side PDF)
research/
  demo.md           ← APR vs AER comparison article (demonstrates cross-referencing)
  aer.md            ← AER research article
  sources/          ← captured source files
docs/               ← generated HTML (gitignored)
```

## Build

- `task build` — build the CLI
- `task test` — run unit tests
- `task check` — fmt + vet + test
- `task docs:build` — render all sources to HTML
- `tp pages` — build and preview docs locally
