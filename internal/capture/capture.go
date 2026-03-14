// Package capture extracts plain text from source documents (HTML, PDF, TXT),
// detects section headings, and produces a source.Source ready to write.
package capture

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"codeberg.org/hum3/cites/internal/source"
	"golang.org/x/net/html"
)

// headingPattern matches lines that look like headings:
// "1. Introduction", "Section 2:", "CHAPTER THREE", or markdown-style "# Heading".
var headingPattern = regexp.MustCompile(
	`^(?:#{1,6}\s+.+|(?:\d+\.)+\s+.+|(?i:chapter|section|part|appendix)\s+.+|[A-Z][A-Z\s]{4,})$`,
)

// CaptureOpts configures optional capture behaviour.
type CaptureOpts struct {
	// PdfSavePath is the path to save the original PDF (only used for PDF sources).
	// If empty, the original PDF is not saved.
	PdfSavePath string
}

// Capture fetches or reads the source at the given location and extracts text.
// location can be a URL or a local file path.
func Capture(location, title string, opts ...CaptureOpts) (*source.Source, error) {
	srcType, body, err := Extract(location)
	if err != nil {
		return nil, err
	}

	sections := detectSections(body, srcType)
	now := time.Now().UTC()
	hash := source.ContentHash(body)

	src := &source.Source{
		Meta: source.Meta{
			Title:         title,
			URL:           canonicalURL(location),
			SourceType:    srcType,
			FirstCaptured: now,
			LastChecked:   now,
			ContentHash:   hash,
			Sections:      sections,
			Versions: []source.Version{
				{
					Date: now.Format("2006-01-02"),
					Hash: hash,
					Note: "Initial capture",
				},
			},
		},
		Body: body,
	}

	// For PDF files, detect page boundaries and assign page anchors to sections.
	if srcType == source.TypePDF {
		var pages []source.PageBreak
		u, _ := url.Parse(location)
		if u != nil && (u.Scheme == "http" || u.Scheme == "https") {
			pages, _ = capturePDFPagesFromURL(location)
		} else {
			pages, _ = CapturePDFPages(location)
		}
		if len(pages) > 0 {
			src.Meta.Pages = pages
			assignPageAnchors(src)
		}

		// Save original PDF if requested.
		var opt CaptureOpts
		if len(opts) > 0 {
			opt = opts[0]
		}
		if opt.PdfSavePath != "" {
			if saveErr := savePDF(location, opt.PdfSavePath); saveErr != nil {
				return nil, fmt.Errorf("saving PDF: %w", saveErr)
			}
			src.Meta.OriginalFile = opt.PdfSavePath
		}
	}

	return src, nil
}

// savePDF copies a PDF from location (URL or local file) to dst.
func savePDF(location, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	u, _ := url.Parse(location)
	if u != nil && (u.Scheme == "http" || u.Scheme == "https") {
		resp, err := http.Get(location)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, resp.Body)
		return err
	}
	// Local file — copy.
	data, err := os.ReadFile(location)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// Extract determines the source type and extracts plain text.
func Extract(location string) (source.SourceType, string, error) {
	u, err := url.Parse(location)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return extractFromURL(location)
	}
	return extractFromFile(location)
}

func extractFromURL(rawURL string) (source.SourceType, string, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("fetching %s: status %d", rawURL, resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "text/html"):
		text, err := extractHTML(resp.Body)
		return source.TypeHTML, text, err
	case strings.Contains(ct, "application/pdf"):
		text, err := extractPDFFromReader(resp.Body)
		return source.TypePDF, text, err
	case strings.Contains(ct, "text/plain"):
		b, err := io.ReadAll(resp.Body)
		return source.TypeTXT, strings.TrimSpace(string(b)), err
	default:
		return "", "", fmt.Errorf("unsupported content type: %s", ct)
	}
}

func extractFromFile(path string) (source.SourceType, string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".htm":
		f, err := os.Open(path)
		if err != nil {
			return "", "", err
		}
		defer f.Close()
		text, err := extractHTML(f)
		return source.TypeHTML, text, err
	case ".pdf":
		text, err := extractPDFFromFile(path)
		return source.TypePDF, text, err
	default:
		b, err := os.ReadFile(path)
		return source.TypeTXT, strings.TrimSpace(string(b)), err
	}
}

// extractHTML walks the HTML tree and extracts visible text.
func extractHTML(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript":
				return
			}
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString("\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		// Add extra newline after block elements for readability.
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6",
				"li", "tr", "blockquote", "pre", "section", "article":
				buf.WriteString("\n")
			}
		}
	}
	walk(doc)

	return strings.TrimSpace(buf.String()), nil
}

// extractPDFFromFile uses pdftotext (poppler-utils) to extract text from a PDF file.
// It returns the extracted text with form feed characters removed.
func extractPDFFromFile(path string) (string, error) {
	cmd := exec.Command("pdftotext", "-layout", path, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext %s: %w (is poppler-utils installed?)", path, err)
	}
	// Remove form feed characters (page boundaries are detected separately).
	text := strings.ReplaceAll(string(out), "\f", "")
	return strings.TrimSpace(text), nil
}

// detectPDFPages scans raw pdftotext output for form feed characters
// and returns page boundary metadata.
func detectPDFPages(rawOutput string) []source.PageBreak {
	var pages []source.PageBreak
	page := 1
	lineNum := 1
	pages = append(pages, source.PageBreak{Page: 1, StartLine: 1})

	for _, ch := range rawOutput {
		if ch == '\f' {
			page++
			pages = append(pages, source.PageBreak{Page: page, StartLine: lineNum})
		} else if ch == '\n' {
			lineNum++
		}
	}
	return pages
}

// extractPDFRaw runs pdftotext and returns the raw output (with form feeds intact).
func extractPDFRaw(path string) (string, error) {
	cmd := exec.Command("pdftotext", "-layout", path, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext %s: %w (is poppler-utils installed?)", path, err)
	}
	return string(out), nil
}

// CapturePDFPages extracts page boundaries from a PDF file.
func CapturePDFPages(path string) ([]source.PageBreak, error) {
	raw, err := extractPDFRaw(path)
	if err != nil {
		return nil, err
	}
	return detectPDFPages(raw), nil
}

// capturePDFPagesFromURL downloads a PDF from a URL to a temp file
// and detects page boundaries from the raw pdftotext output.
func capturePDFPagesFromURL(rawURL string) ([]source.PageBreak, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "cites-pages-*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return nil, err
	}
	tmp.Close()

	return CapturePDFPages(tmp.Name())
}

// extractPDFFromReader writes to a temp file then calls pdftotext.
func extractPDFFromReader(r io.Reader) (string, error) {
	tmp, err := os.CreateTemp("", "cites-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, r); err != nil {
		return "", err
	}
	tmp.Close()

	return extractPDFFromFile(tmp.Name())
}

// detectSections scans the extracted text for heading-like lines.
func detectSections(body string, srcType source.SourceType) []source.Section {
	lines := strings.Split(body, "\n")
	var sections []source.Section

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if headingPattern.MatchString(trimmed) {
			sec := source.Section{
				Line:    i + 1, // 1-based
				Heading: trimmed,
			}
			// For HTML sources, generate a slug-based original anchor.
			if srcType == source.TypeHTML {
				sec.OriginalAnchor = "#" + slugify(trimmed)
			}
			// For PDFs, we can't reliably generate original anchors
			// without page number mapping — leave blank.
			sections = append(sections, sec)
		}
	}

	return sections
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// assignPageAnchors sets the OriginalAnchor on each section to #page=N
// based on the page boundaries.
func assignPageAnchors(src *source.Source) {
	for i := range src.Meta.Sections {
		page := src.PageForLine(src.Meta.Sections[i].Line)
		if page > 0 {
			src.Meta.Sections[i].OriginalAnchor = fmt.Sprintf("#page=%d", page)
		}
	}
}

func canonicalURL(location string) string {
	u, err := url.Parse(location)
	if err != nil || u.Scheme == "" {
		return "" // local file — no canonical URL
	}
	return location
}
