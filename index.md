# cites

Citation reference tooling: capture source documents and render HTML reference pages
with line-numbered, anchor-linked text. Version **__VERSION__**.

## Documentation

| Ref | Source | Type | Description |
|-----|--------|------|-------------|
| — | [cites](https://codeberg.org/hum3/cites) | — | Source code and project documentation |
| [r0](r0/) | [Sample HTML](r0/) | HTML | Test capture: headings, script/style filtering |
| [r1](r1/) | [Sample TXT](r1/) | TXT | Test capture: numbered headings, chapter markers |
| [r2](r2/) | [AER Practice Note](r2/) | PDF | UK Finance AER calculation methodology, Jan 2025 |
| r3 | Consumer Credit Act 1974 | HTML | Full text from legislation.gov.uk |

## How it works

### 1. Capture a source document

Download and extract plain text, detect headings, compute a content hash:

```
cites capture https://www.legislation.gov.uk/ukpga/1974/39 \
  "Consumer Credit Act 1974" research/sources/cca-1974.yaml
```

This creates `research/sources/cca-1974.yaml` containing YAML frontmatter (title, URL,
hash, sections) followed by the extracted plain text.

To check whether the upstream source has changed since capture:

```
cites check research/sources/cca-1974.yaml
```

Add `--update` to replace the body and hash if the source has changed.

### 2. Render to HTML

Generate a line-numbered reference page in the docs folder:

```
cites render --base .. research/sources/cca-1974.yaml docs/r3
```

The rendered page includes:

- **Metadata block** — title, source URL, capture/check dates, content hash
- **Section index** — detected headings with line-number links
- **Version history** — hash snapshots for change tracking
- **Line-numbered text** — every line has a clickable anchor (`#l42`)
- **Paginated output** — one HTML page per PDF page (or ~100 lines for HTML/TXT)
- **PDF thumbnails** — side-by-side layout with page thumbnails when original PDF available

### 3. Reference specific lines

Once rendered, cite any passage by line number:

```markdown
See [CCA s.20 — Total charge for credit](r3/p03.html#l142)
```

From a research article, link into the captured source files directly:

```markdown
The [AER Practice Note](sources/aer-practice-note-2025.yaml#L1) defines the scope…
```

Or link to the rendered HTML for readers:

```markdown
- [Definition and scope](r2/p01.html#l1)
- [Sample HTML line 6](r0/p01.html#l6) — "Section 2: Details"
- [Sample TXT line 9](r1/p01.html#l9) — "CHAPTER THREE"
```

## Demos

### APR vs AER comparison

The [demo article](research/demo.md) compares two UK interest rate standards by
cross-referencing captured sources:

- **Consumer Credit Act 1974** — [s.20 Total charge for credit](r3/p03.html#l142),
  [Part V Entry into agreements](r3/p02.html#l200)
- **AER Practice Note** — [Definition and scope](r2/p01.html#l1)

### Research articles

- [APR vs AER comparison](research/demo.html)
- [AER — Annual Equivalent Rate](research/aer.html)

### Source reference pages

- [r2 — AER Practice Note](r2/) — PDF with paginated page navigation
- [r3 — Consumer Credit Act 1974](r3/) — HTML extraction with section detection

## Source

- [codeberg.org/hum3/cites](https://codeberg.org/hum3/cites)
- [github.com/drummonds/cites](https://github.com/drummonds/cites) (mirror)
