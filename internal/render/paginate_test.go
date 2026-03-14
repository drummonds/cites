package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/hum3/cites/internal/source"
)

func TestPagesForSourcePDF(t *testing.T) {
	src := newTestSource(source.TypePDF)
	src.Meta.Pages = []source.PageBreak{
		{Page: 1, StartLine: 1},
		{Page: 2, StartLine: 4},
	}

	pages := PagesForSource(src, 100)
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].File != "p01.html" {
		t.Errorf("page 1 file: got %q", pages[0].File)
	}
	if pages[0].StartLine != 1 || pages[0].EndLine != 3 {
		t.Errorf("page 1 lines: start=%d end=%d", pages[0].StartLine, pages[0].EndLine)
	}
	if pages[1].StartLine != 4 || pages[1].EndLine != 5 {
		t.Errorf("page 2 lines: start=%d end=%d", pages[1].StartLine, pages[1].EndLine)
	}
}

func TestPagesForSourceTXT(t *testing.T) {
	src := newTestSource(source.TypeTXT)
	// 5 lines, 3 per page
	pages := PagesForSource(src, 3)
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].StartLine != 1 || pages[0].EndLine != 3 {
		t.Errorf("page 1: start=%d end=%d", pages[0].StartLine, pages[0].EndLine)
	}
	if pages[1].StartLine != 4 || pages[1].EndLine != 5 {
		t.Errorf("page 2: start=%d end=%d", pages[1].StartLine, pages[1].EndLine)
	}
}

func TestRenderDir(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	outDir := t.TempDir()

	opts := RenderOpts{
		LinesPerPage: 3,
		BasePath:     "..",
	}

	if err := RenderDir(outDir, src, opts); err != nil {
		t.Fatalf("RenderDir: %v", err)
	}

	// Check index.html exists and has key content.
	indexBytes, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("reading index.html: %v", err)
	}
	index := string(indexBytes)
	for _, want := range []string{"Test Document", "p01.html", "p02.html", "Intro", "Details"} {
		if !strings.Contains(index, want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	// Check page files exist.
	for _, f := range []string{"p01.html", "p02.html"} {
		pageBytes, err := os.ReadFile(filepath.Join(outDir, f))
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		page := string(pageBytes)
		if !strings.Contains(page, "Test Document") {
			t.Errorf("%s missing title", f)
		}
		if !strings.Contains(page, "cites") {
			t.Errorf("%s missing breadcrumb", f)
		}
	}

	// Page 1 should have line anchors l1, l2, l3.
	p1, _ := os.ReadFile(filepath.Join(outDir, "p01.html"))
	if !strings.Contains(string(p1), `id="l1"`) {
		t.Error("p01.html missing line anchor l1")
	}
	if !strings.Contains(string(p1), `id="l3"`) {
		t.Error("p01.html missing line anchor l3")
	}

	// Page 2 should have l4, l5.
	p2, _ := os.ReadFile(filepath.Join(outDir, "p02.html"))
	if !strings.Contains(string(p2), `id="l4"`) {
		t.Error("p02.html missing line anchor l4")
	}

	// Navigation: page 1 has no prev, has next; page 2 has prev, no next.
	if strings.Contains(string(p1), `← Previous</a>`) {
		t.Error("p01 should not have prev link")
	}
	if !strings.Contains(string(p1), `Next →</a>`) {
		t.Error("p01 should have next link")
	}
	if !strings.Contains(string(p2), `← Previous</a>`) {
		t.Error("p02 should have prev link")
	}
}

func TestRenderDirPDFSideBySide(t *testing.T) {
	src := newTestSource(source.TypePDF)
	src.Meta.Pages = []source.PageBreak{
		{Page: 1, StartLine: 1},
		{Page: 2, StartLine: 4},
	}

	outDir := t.TempDir()
	opts := RenderOpts{BasePath: ".."}

	if err := RenderDir(outDir, src, opts); err != nil {
		t.Fatalf("RenderDir: %v", err)
	}

	// Without PDF path, should not have thumbnails — no side-layout div in body.
	p1, _ := os.ReadFile(filepath.Join(outDir, "p01.html"))
	if strings.Contains(string(p1), `<div class="side-layout">`) {
		t.Error("without --pdf, should not have side layout div")
	}
}

func TestIndexCombinedTOC(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	outDir := t.TempDir()

	opts := RenderOpts{LinesPerPage: 3, BasePath: ".."}
	if err := RenderDir(outDir, src, opts); err != nil {
		t.Fatalf("RenderDir: %v", err)
	}

	indexBytes, _ := os.ReadFile(filepath.Join(outDir, "index.html"))
	index := string(indexBytes)

	// Combined TOC should have page rows and section rows.
	if !strings.Contains(index, "page-row") {
		t.Error("index should have page rows in TOC")
	}
	if !strings.Contains(index, "section-row") {
		t.Error("index should have section rows in TOC")
	}
	// Section links: "Intro" at line 1 → p01.html#l1
	if !strings.Contains(index, `p01.html#l1`) {
		t.Error("index should link Intro to p01.html#l1")
	}
	// Section "Details" at line 4 → p02.html#l4
	if !strings.Contains(index, `p02.html#l4`) {
		t.Error("index should link Details to p02.html#l4")
	}
	// Page links with line ranges.
	if !strings.Contains(index, "Page 1") {
		t.Error("index should show Page 1")
	}
	if !strings.Contains(index, "Page 2") {
		t.Error("index should show Page 2")
	}
}

func TestIndexVersionHighlight(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	outDir := t.TempDir()

	opts := RenderOpts{LinesPerPage: 3, BasePath: ".."}
	if err := RenderDir(outDir, src, opts); err != nil {
		t.Fatalf("RenderDir: %v", err)
	}

	indexBytes, _ := os.ReadFile(filepath.Join(outDir, "index.html"))
	index := string(indexBytes)

	// Current version should be highlighted.
	if !strings.Contains(index, "version-current") {
		t.Error("index should highlight current version")
	}
	if !strings.Contains(index, "(current)") {
		t.Error("index should mark current version with (current)")
	}
}
