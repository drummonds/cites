// Package source defines the captured source file format:
// YAML frontmatter separated by "---" lines, followed by plain text content.
package source

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SourceType is the format of the original document.
type SourceType string

const (
	TypePDF  SourceType = "pdf"
	TypeHTML SourceType = "html"
	TypeTXT  SourceType = "txt"
)

// Section maps a heading in the extracted text to its original anchor.
type Section struct {
	Line           int    `yaml:"line"`
	Heading        string `yaml:"heading"`
	OriginalAnchor string `yaml:"original_anchor,omitempty"`
}

// PageBreak records a PDF page boundary: which page starts at which line.
type PageBreak struct {
	Page      int `yaml:"page"`
	StartLine int `yaml:"start_line"`
}

// Version records a snapshot of the source content at a point in time.
type Version struct {
	Date string `yaml:"date"`
	Hash string `yaml:"hash"`
	Note string `yaml:"note,omitempty"`
}

// Meta is the YAML frontmatter for a captured source file.
type Meta struct {
	Title         string      `yaml:"title"`
	URL           string      `yaml:"url"`
	SourceType    SourceType  `yaml:"source_type"`
	SourceDate    string      `yaml:"source_date,omitempty"`
	FirstCaptured time.Time   `yaml:"first_captured"`
	LastChecked   time.Time   `yaml:"last_checked"`
	ContentHash   string      `yaml:"content_hash"`
	Sections      []Section   `yaml:"sections,omitempty"`
	Pages         []PageBreak `yaml:"pages,omitempty"`
	Versions      []Version   `yaml:"versions,omitempty"`
}

// Source is a captured source document: metadata + extracted text.
type Source struct {
	Meta Meta
	Body string // plain text content (no frontmatter)
}

// ContentHash computes the SHA-256 hash of the body text.
func ContentHash(body string) string {
	h := sha256.Sum256([]byte(body))
	return fmt.Sprintf("sha256:%x", h)
}

// HasChanged returns true if the body's current hash differs from the stored hash.
func (s *Source) HasChanged() bool {
	return ContentHash(s.Body) != s.Meta.ContentHash
}

// SectionForLine returns the closest section at or before the given line number.
func (s *Source) SectionForLine(line int) *Section {
	var best *Section
	for i := range s.Meta.Sections {
		if s.Meta.Sections[i].Line <= line {
			best = &s.Meta.Sections[i]
		}
	}
	return best
}

// PageForLine returns the page number for the given line, or 0 if no pages recorded.
func (s *Source) PageForLine(line int) int {
	page := 0
	for _, p := range s.Meta.Pages {
		if p.StartLine <= line {
			page = p.Page
		}
	}
	return page
}

// Parse reads a source file (frontmatter + body) from r.
func Parse(r io.Reader) (*Source, error) {
	scanner := bufio.NewScanner(r)
	var (
		inFront   bool
		front     []string
		bodyLines []string
		pastFront bool
	)

	for scanner.Scan() {
		line := scanner.Text()
		if !inFront && !pastFront && strings.TrimSpace(line) == "---" {
			inFront = true
			continue
		}
		if inFront && strings.TrimSpace(line) == "---" {
			inFront = false
			pastFront = true
			continue
		}
		if inFront {
			front = append(front, line)
		} else if pastFront {
			bodyLines = append(bodyLines, line)
		} else {
			// No frontmatter — treat entire file as body.
			bodyLines = append(bodyLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading source: %w", err)
	}

	src := &Source{
		Body: strings.Join(bodyLines, "\n"),
	}

	if len(front) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(front, "\n")), &src.Meta); err != nil {
			return nil, fmt.Errorf("parsing frontmatter: %w", err)
		}
	}

	return src, nil
}

// ParseFile reads a source file from disk.
func ParseFile(path string) (*Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Marshal serializes a Source back to frontmatter + body format.
func (s *Source) Marshal() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&s.Meta); err != nil {
		return nil, fmt.Errorf("encoding frontmatter: %w", err)
	}
	enc.Close()
	buf.WriteString("---\n")
	buf.WriteString(s.Body)

	return buf.Bytes(), nil
}

// WriteFile writes the source to disk.
func (s *Source) WriteFile(path string) error {
	data, err := s.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
