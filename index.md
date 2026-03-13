# cites

Citation reference tooling: capture source documents and render HTML reference pages
with line-numbered, anchor-linked text.

## How it works

1. **Capture** a source document (HTML, PDF, or plain text) — extracts plain text with metadata
2. **Render** the captured source to an HTML reference page with line anchors (`#L42`)
3. **Check** whether the upstream source has changed since capture

Each rendered page shows the extracted text with line numbers. Every line is addressable
via URL fragment (e.g. `sample-html-test.html#L6`) making it easy to cite specific
passages from research articles.

## Test cases

These test captures demonstrate the rendering pipeline:

| Source | Type | Description |
|--------|------|-------------|
| [Sample HTML](sample-html-test.html) | HTML | Minimal HTML with headings, script/style filtering |
| [Sample TXT](sample-txt-test.html) | TXT | Plain text with numbered headings, chapter markers |

### Referencing lines

Once rendered, any line can be linked to directly. For example:

- [Sample HTML line 6](sample-html-test.html#L6) — "Section 2: Details" heading
- [Sample HTML line 14](sample-html-test.html#L14) — "APPENDIX A" heading
- [Sample TXT line 1](sample-txt-test.html#L1) — "1. Introduction"
- [Sample TXT line 9](sample-txt-test.html#L9) — "CHAPTER THREE"

### What the rendered page shows

Each reference page includes:

- **Metadata block** — title, source URL, capture/check dates, content hash
- **Section index** — detected headings with line-number links
- **Version history** — hash snapshots for change tracking
- **Line-numbered text** — every line has a clickable anchor

For PDF sources with page boundary data, the layout switches to a **side-by-side view**
with page navigation on the left linking to the original PDF.

## Source

- [codeberg.org/hum3/cites](https://codeberg.org/hum3/cites)
- [github.com/drummonds/cites](https://github.com/drummonds/cites) (mirror)
