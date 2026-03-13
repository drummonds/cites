package source

import (
	"strings"
	"testing"
	"time"
)

func TestContentHash(t *testing.T) {
	h := ContentHash("hello")
	if !strings.HasPrefix(h, "sha256:") {
		t.Fatalf("expected sha256: prefix, got %s", h)
	}
	if ContentHash("hello") != ContentHash("hello") {
		t.Fatal("same input should produce same hash")
	}
	if ContentHash("hello") == ContentHash("world") {
		t.Fatal("different input should produce different hash")
	}
}

func TestHasChanged(t *testing.T) {
	body := "test content"
	src := &Source{
		Meta: Meta{ContentHash: ContentHash(body)},
		Body: body,
	}
	if src.HasChanged() {
		t.Fatal("should not be changed when hash matches")
	}
	src.Body = "modified content"
	if !src.HasChanged() {
		t.Fatal("should be changed when body differs")
	}
}

func TestSectionForLine(t *testing.T) {
	src := &Source{
		Meta: Meta{
			Sections: []Section{
				{Line: 1, Heading: "Intro"},
				{Line: 10, Heading: "Methods"},
				{Line: 20, Heading: "Results"},
			},
		},
	}

	tests := []struct {
		line    int
		heading string
	}{
		{1, "Intro"},
		{5, "Intro"},
		{10, "Methods"},
		{15, "Methods"},
		{20, "Results"},
		{100, "Results"},
	}

	for _, tt := range tests {
		sec := src.SectionForLine(tt.line)
		if sec == nil {
			t.Fatalf("line %d: expected section, got nil", tt.line)
		}
		if sec.Heading != tt.heading {
			t.Errorf("line %d: got %q, want %q", tt.line, sec.Heading, tt.heading)
		}
	}

	// Line before any section
	src2 := &Source{Meta: Meta{Sections: []Section{{Line: 5, Heading: "A"}}}}
	if sec := src2.SectionForLine(3); sec != nil {
		t.Errorf("line 3: expected nil, got %v", sec)
	}
}

func TestPageForLine(t *testing.T) {
	src := &Source{
		Meta: Meta{
			Pages: []PageBreak{
				{Page: 1, StartLine: 1},
				{Page: 2, StartLine: 10},
				{Page: 3, StartLine: 20},
			},
		},
	}

	tests := []struct {
		line int
		page int
	}{
		{1, 1},
		{5, 1},
		{10, 2},
		{15, 2},
		{20, 3},
		{100, 3},
	}

	for _, tt := range tests {
		p := src.PageForLine(tt.line)
		if p != tt.page {
			t.Errorf("line %d: got page %d, want %d", tt.line, p, tt.page)
		}
	}

	// No pages
	src2 := &Source{}
	if p := src2.PageForLine(5); p != 0 {
		t.Errorf("no pages: got %d, want 0", p)
	}
}

func TestParseAndMarshalRoundTrip(t *testing.T) {
	now := time.Date(2025, 3, 13, 10, 30, 0, 0, time.UTC)
	original := &Source{
		Meta: Meta{
			Title:         "Test Document",
			URL:           "https://example.com/test.pdf",
			SourceType:    TypePDF,
			SourceDate:    "2024-01-15",
			FirstCaptured: now,
			LastChecked:   now,
			ContentHash:   "sha256:abc123",
			Sections: []Section{
				{Line: 1, Heading: "Intro", OriginalAnchor: "#page=1"},
				{Line: 10, Heading: "Body"},
			},
			Pages: []PageBreak{
				{Page: 1, StartLine: 1},
				{Page: 2, StartLine: 10},
			},
			Versions: []Version{
				{Date: "2025-03-13", Hash: "sha256:abc123", Note: "Initial"},
			},
		},
		Body: "Line one\nLine two\nLine three",
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	parsed, err := Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if parsed.Meta.Title != original.Meta.Title {
		t.Errorf("title: got %q, want %q", parsed.Meta.Title, original.Meta.Title)
	}
	if parsed.Meta.URL != original.Meta.URL {
		t.Errorf("url: got %q, want %q", parsed.Meta.URL, original.Meta.URL)
	}
	if parsed.Meta.SourceType != original.Meta.SourceType {
		t.Errorf("source_type: got %q, want %q", parsed.Meta.SourceType, original.Meta.SourceType)
	}
	if parsed.Body != original.Body {
		t.Errorf("body: got %q, want %q", parsed.Body, original.Body)
	}
	if len(parsed.Meta.Sections) != len(original.Meta.Sections) {
		t.Errorf("sections: got %d, want %d", len(parsed.Meta.Sections), len(original.Meta.Sections))
	}
	if len(parsed.Meta.Pages) != len(original.Meta.Pages) {
		t.Errorf("pages: got %d, want %d", len(parsed.Meta.Pages), len(original.Meta.Pages))
	}
	if len(parsed.Meta.Versions) != len(original.Meta.Versions) {
		t.Errorf("versions: got %d, want %d", len(parsed.Meta.Versions), len(original.Meta.Versions))
	}
}

func TestParseFile(t *testing.T) {
	src, err := ParseFile("testdata/sample_source.txt")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if src.Meta.Title != "Test Source Document" {
		t.Errorf("title: got %q", src.Meta.Title)
	}
	if src.Meta.SourceType != TypePDF {
		t.Errorf("source_type: got %q", src.Meta.SourceType)
	}
	if len(src.Meta.Sections) != 2 {
		t.Errorf("sections: got %d, want 2", len(src.Meta.Sections))
	}
	if len(src.Meta.Pages) != 2 {
		t.Errorf("pages: got %d, want 2", len(src.Meta.Pages))
	}
	if src.HasChanged() {
		t.Error("should not be changed — hash should match body")
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	input := "Just plain text\nwith no frontmatter."
	src, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if src.Body != input {
		t.Errorf("body: got %q, want %q", src.Body, input)
	}
	if src.Meta.Title != "" {
		t.Errorf("title should be empty, got %q", src.Meta.Title)
	}
}
