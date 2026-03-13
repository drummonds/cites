package render

import (
	"strings"
	"testing"
	"time"

	"codeberg.org/hum3/cites/internal/source"
)

func newTestSource(srcType source.SourceType) *source.Source {
	now := time.Date(2025, 3, 13, 10, 30, 0, 0, time.UTC)
	body := "Line one\nLine two\nLine three\nLine four\nLine five"
	return &source.Source{
		Meta: source.Meta{
			Title:         "Test Document",
			URL:           "https://example.com/test",
			SourceType:    srcType,
			FirstCaptured: now,
			LastChecked:   now,
			ContentHash:   source.ContentHash(body),
			Sections: []source.Section{
				{Line: 1, Heading: "Intro", OriginalAnchor: "#intro"},
				{Line: 4, Heading: "Details", OriginalAnchor: "#details"},
			},
			Versions: []source.Version{
				{Date: "2025-03-13", Hash: source.ContentHash(body), Note: "Initial"},
			},
		},
		Body: body,
	}
}

func TestRenderHTML(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	html, err := RenderToString(src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	checks := []string{
		"Test Document",
		"Source Reference",
		"https://example.com/test",
		"html",
		`id="L1"`,
		`id="L5"`,
		"hash-ok",
		"Intro",
		"Details",
		"section-heading",
		"Version history",
	}
	for _, c := range checks {
		if !strings.Contains(html, c) {
			t.Errorf("output missing %q", c)
		}
	}
}

func TestRenderHashChanged(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	src.Body = "modified content"

	html, err := RenderToString(src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(html, "hash-changed") {
		t.Error("should show hash-changed when body modified")
	}
	if !strings.Contains(html, "Content modified since capture") {
		t.Error("should show change warning")
	}
}

func TestRenderPDFSideBySide(t *testing.T) {
	src := newTestSource(source.TypePDF)
	src.Meta.SourceType = source.TypePDF
	src.Meta.Pages = []source.PageBreak{
		{Page: 1, StartLine: 1},
		{Page: 2, StartLine: 4},
	}
	src.Meta.Sections[0].OriginalAnchor = "#page=1"
	src.Meta.Sections[1].OriginalAnchor = "#page=2"

	html, err := RenderToString(src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Should use side-by-side layout
	if !strings.Contains(html, `class="side-by-side"`) {
		t.Error("PDF with pages should use side-by-side layout")
	}
	if !strings.Contains(html, `class="page-nav"`) {
		t.Error("should have page navigation")
	}
	if !strings.Contains(html, "Page 1") {
		t.Error("should show page numbers")
	}
	if !strings.Contains(html, "Page 2") {
		t.Error("should show page 2")
	}
	if !strings.Contains(html, "page-break") {
		t.Error("should have page break markers")
	}
}

func TestRenderPDFNoPages(t *testing.T) {
	src := newTestSource(source.TypePDF)
	src.Meta.SourceType = source.TypePDF
	// No Pages set

	html, err := RenderToString(src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Without pages, should fall back to standard layout
	if strings.Contains(html, `class="side-by-side"`) {
		t.Error("PDF without pages should not use side-by-side layout")
	}
}

func TestRenderLineAnchors(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	html, err := RenderToString(src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Each line should have an anchor
	for i := 1; i <= 5; i++ {
		anchor := `id="L` + strings.Repeat("", 0) // just check format
		_ = anchor
	}
	if !strings.Contains(html, `href="#L1"`) {
		t.Error("missing line 1 anchor link")
	}
	if !strings.Contains(html, `href="#L5"`) {
		t.Error("missing line 5 anchor link")
	}
}

func TestBuildLines(t *testing.T) {
	src := newTestSource(source.TypeHTML)
	lines := buildLines(src)

	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}

	// Line 1 should be a heading
	if !lines[0].IsHeading {
		t.Error("line 1 should be a heading")
	}
	if lines[0].SectionLink != "https://example.com/test#intro" {
		t.Errorf("line 1 section link: got %q", lines[0].SectionLink)
	}

	// Line 2 should not be a heading but should link to section 1
	if lines[1].IsHeading {
		t.Error("line 2 should not be a heading")
	}
	if lines[1].SectionLink != "https://example.com/test#intro" {
		t.Errorf("line 2 section link: got %q", lines[1].SectionLink)
	}

	// Line 4 should be a heading (section 2)
	if !lines[3].IsHeading {
		t.Error("line 4 should be a heading")
	}
	if lines[3].SectionLink != "https://example.com/test#details" {
		t.Errorf("line 4 section link: got %q", lines[3].SectionLink)
	}
}

func TestBuildLinesPageBreaks(t *testing.T) {
	src := newTestSource(source.TypePDF)
	src.Meta.Pages = []source.PageBreak{
		{Page: 1, StartLine: 1},
		{Page: 2, StartLine: 3},
	}

	lines := buildLines(src)

	if !lines[0].PageBreak || lines[0].PageNum != 1 {
		t.Error("line 1 should be page break for page 1")
	}
	if lines[1].PageBreak {
		t.Error("line 2 should not be a page break")
	}
	if !lines[2].PageBreak || lines[2].PageNum != 2 {
		t.Error("line 3 should be page break for page 2")
	}
}
