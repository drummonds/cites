package capture

import (
	"os"
	"strings"
	"testing"

	"codeberg.org/hum3/cites/internal/source"
)

func TestExtractHTML(t *testing.T) {
	f, err := os.Open("testdata/sample.html")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	text, err := extractHTML(f)
	if err != nil {
		t.Fatalf("extractHTML: %v", err)
	}

	// Should contain visible text.
	if !strings.Contains(text, "Introduction") {
		t.Error("missing 'Introduction'")
	}
	if !strings.Contains(text, "first paragraph") {
		t.Error("missing 'first paragraph'")
	}
	if !strings.Contains(text, "APPENDIX A") {
		t.Error("missing 'APPENDIX A'")
	}

	// Should NOT contain script/style/noscript content.
	if strings.Contains(text, "var x = 1") {
		t.Error("should not contain script content")
	}
	if strings.Contains(text, ".hidden") {
		t.Error("should not contain style content")
	}
	if strings.Contains(text, "Enable JavaScript") {
		t.Error("should not contain noscript content")
	}
}

func TestDetectSections(t *testing.T) {
	body := `1. Introduction

Some text here.

2. Background

More text.

CHAPTER THREE

Content here.

Section 4: Conclusion

Final text.`

	sections := detectSections(body, source.TypeTXT)
	if len(sections) < 4 {
		t.Fatalf("expected at least 4 sections, got %d", len(sections))
	}

	// Check first section
	if sections[0].Heading != "1. Introduction" {
		t.Errorf("section 0: got %q", sections[0].Heading)
	}
	if sections[0].Line != 1 {
		t.Errorf("section 0 line: got %d", sections[0].Line)
	}

	// TXT should not have original anchors
	for _, s := range sections {
		if s.OriginalAnchor != "" {
			t.Errorf("TXT section should not have anchor, got %q", s.OriginalAnchor)
		}
	}
}

func TestDetectSectionsHTML(t *testing.T) {
	body := "# Welcome\n\nSome content.\n\n## Details\n\nMore content."
	sections := detectSections(body, source.TypeHTML)

	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	// HTML sections should have slug-based anchors
	if sections[0].OriginalAnchor != "#welcome" {
		t.Errorf("section 0 anchor: got %q, want '#welcome'", sections[0].OriginalAnchor)
	}
	if sections[1].OriginalAnchor != "#details" {
		t.Errorf("section 1 anchor: got %q, want '#details'", sections[1].OriginalAnchor)
	}
}

func TestCaptureLocalTXT(t *testing.T) {
	src, err := Capture("testdata/sample.txt", "Sample Text")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	if src.Meta.Title != "Sample Text" {
		t.Errorf("title: got %q", src.Meta.Title)
	}
	if src.Meta.SourceType != source.TypeTXT {
		t.Errorf("type: got %q", src.Meta.SourceType)
	}
	if src.Meta.ContentHash == "" {
		t.Error("content hash should not be empty")
	}
	if src.Meta.URL != "" {
		t.Error("local file should have empty URL")
	}
	if len(src.Meta.Sections) == 0 {
		t.Error("expected at least one section")
	}
	if len(src.Meta.Versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(src.Meta.Versions))
	}
	if src.Meta.Versions[0].Note != "Initial capture" {
		t.Errorf("version note: got %q", src.Meta.Versions[0].Note)
	}
}

func TestCaptureLocalHTML(t *testing.T) {
	src, err := Capture("testdata/sample.html", "Sample HTML")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	if src.Meta.SourceType != source.TypeHTML {
		t.Errorf("type: got %q", src.Meta.SourceType)
	}
	if !strings.Contains(src.Body, "Introduction") {
		t.Error("body should contain 'Introduction'")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Hello World", "hello-world"},
		{"Section 2: Details", "section-2-details"},
		{"# Heading", "heading"},
		{"CHAPTER THREE", "chapter-three"},
		{"  spaces  ", "spaces"},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectPDFPages(t *testing.T) {
	// Simulate pdftotext output with form feeds
	raw := "Page 1 line 1\nPage 1 line 2\n\fPage 2 line 1\nPage 2 line 2\n\fPage 3 line 1\n"
	pages := detectPDFPages(raw)

	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
	if pages[0].Page != 1 || pages[0].StartLine != 1 {
		t.Errorf("page 1: got page=%d start=%d", pages[0].Page, pages[0].StartLine)
	}
	if pages[1].Page != 2 || pages[1].StartLine != 3 {
		t.Errorf("page 2: got page=%d start=%d", pages[1].Page, pages[1].StartLine)
	}
	if pages[2].Page != 3 || pages[2].StartLine != 5 {
		t.Errorf("page 3: got page=%d start=%d", pages[2].Page, pages[2].StartLine)
	}
}

func TestCanonicalURL(t *testing.T) {
	if u := canonicalURL("https://example.com/test"); u != "https://example.com/test" {
		t.Errorf("URL: got %q", u)
	}
	if u := canonicalURL("testdata/sample.txt"); u != "" {
		t.Errorf("local file: got %q, want empty", u)
	}
}
