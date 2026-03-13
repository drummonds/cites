# Roadmap

## v0.1.0

- [x] Source file format with YAML frontmatter
- [x] Text extraction from HTML, PDF, and TXT sources
- [x] Section heading detection with original anchor mapping
- [x] PDF page boundary detection with `pages` metadata
- [x] Side-by-side PDF render layout with page navigation
- [x] SHA-256 content hashing and change detection
- [x] Line-numbered HTML reference page rendering
- [x] Check command with re-fetch, version tracking, and `--update` flag
- [x] Unit tests for source, capture, and render packages
- [x] `docs:build` task wired up
- [ ] Capture AER Practice Note (real-world PDF test case)
- [ ] Capture Consumer Credit Act 1974 (real-world HTML test case)
- [ ] Render demo article with citation link resolution

## Future

- [ ] Citation index generation (cross-reference map across all articles)
- [ ] Article rendering (Markdown → HTML with citation link resolution)
- [ ] Static site deployment via statichost
