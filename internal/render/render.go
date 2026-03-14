// Package render generates HTML reference documents from captured sources.
// Each rendered page has a metadata header block and line-numbered text
// with anchors and links back to the closest section in the original source.
//
// For PDF sources with page data, a side-by-side layout is used:
// left panel links to specific pages in the source PDF,
// right panel shows line-numbered extracted text.
package render

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	"codeberg.org/hum3/cites/internal/source"
)

var refTemplate = template.Must(template.New("reference").Funcs(template.FuncMap{
	"add":   func(a, b int) int { return a + b },
	"isPDF": func(st source.SourceType) bool { return st == source.TypePDF },
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Meta.Title }} — Source Reference</title>
  <style>
    :root { --bg: #fff; --fg: #1a1a1a; --muted: #666; --border: #ddd; --link: #0366d6; --line-bg: #f6f8fa; }
    @media (prefers-color-scheme: dark) {
      :root { --bg: #0d1117; --fg: #c9d1d9; --muted: #8b949e; --border: #30363d; --link: #58a6ff; --line-bg: #161b22; }
    }
    body { font-family: system-ui, sans-serif; max-width: 960px; margin: 2rem auto; padding: 0 1rem; background: var(--bg); color: var(--fg); }
    .meta { border: 1px solid var(--border); border-radius: 6px; padding: 1rem 1.5rem; margin-bottom: 2rem; background: var(--line-bg); }
    .meta h1 { margin: 0 0 0.5rem; font-size: 1.4rem; }
    .meta dl { display: grid; grid-template-columns: auto 1fr; gap: 0.25rem 1rem; margin: 0; }
    .meta dt { font-weight: 600; color: var(--muted); }
    .meta dd { margin: 0; }
    .meta a { color: var(--link); }
    .hash-ok { color: #2ea043; }
    .hash-changed { color: #d73a49; font-weight: bold; }
    .sections { margin-bottom: 2rem; }
    .sections summary { cursor: pointer; font-weight: 600; }
    .sections ul { list-style: none; padding-left: 1rem; }
    .sections a { color: var(--link); text-decoration: none; }
    .source-text { font-family: 'SFMono-Regular', Consolas, monospace; font-size: 0.85rem; line-height: 1.6; }
    .source-line { display: flex; }
    .source-line:target { background: #fff8c5; }
    @media (prefers-color-scheme: dark) { .source-line:target { background: #3b2e00; } }
    .line-num { flex: 0 0 5ch; text-align: right; padding-right: 1ch; color: var(--muted); user-select: none; }
    .line-num a { color: inherit; text-decoration: none; }
    .line-num a:hover { color: var(--link); }
    .line-text { flex: 1; white-space: pre-wrap; word-break: break-word; }
    .section-heading { font-weight: bold; color: var(--link); }
    /* Side-by-side layout for PDF sources */
    .side-by-side { display: grid; grid-template-columns: 180px 1fr; gap: 1rem; }
    .page-nav { position: sticky; top: 1rem; align-self: start; max-height: calc(100vh - 2rem); overflow-y: auto; }
    .page-nav ul { list-style: none; padding: 0; margin: 0; }
    .page-nav li { margin-bottom: 0.25rem; }
    .page-nav a { display: block; padding: 0.25rem 0.5rem; color: var(--link); text-decoration: none; border-radius: 4px; }
    .page-nav a:hover { background: var(--line-bg); }
    .page-nav .active { background: var(--line-bg); font-weight: 600; }
    .page-break { border-top: 2px dashed var(--border); margin: 0.5rem 0; padding-top: 0.25rem; }
    .topbar { background: var(--link); color: #fff; padding: 0.5rem 1.5rem; margin-bottom: 1.5rem; border-radius: 6px; font-size: 0.85rem; }
    .topbar a { color: #fff; text-decoration: none; font-weight: 600; }
    .topbar a:hover { text-decoration: underline; }
    @media (max-width: 700px) { .side-by-side { grid-template-columns: 1fr; } .page-nav { display: none; } }
  </style>
</head>
<body>

<nav class="topbar"><a href="{{ if .BasePath }}{{ .BasePath }}/{{ end }}index.html">cites</a> › {{ .Meta.Title }}</nav>

<div class="meta">
  <h1>{{ .Meta.Title }}</h1>
  <dl>
    {{ if .Meta.URL }}<dt>Source</dt><dd><a href="{{ .Meta.URL }}">{{ .Meta.URL }}</a></dd>{{ end }}
    <dt>Type</dt><dd>{{ .Meta.SourceType }}</dd>
    {{ if .Meta.SourceDate }}<dt>Source date</dt><dd>{{ .Meta.SourceDate }}</dd>{{ end }}
    <dt>First captured</dt><dd>{{ .Meta.FirstCaptured.Format "2006-01-02 15:04 UTC" }}</dd>
    <dt>Last checked</dt><dd>{{ .Meta.LastChecked.Format "2006-01-02 15:04 UTC" }}</dd>
    <dt>Content hash</dt><dd><code{{ if .HashChanged }} class="hash-changed" title="Content has changed since capture"{{ else }} class="hash-ok"{{ end }}>{{ .Meta.ContentHash }}</code></dd>
    {{ if .HashChanged }}<dt>Status</dt><dd class="hash-changed">⚠ Content modified since capture</dd>{{ end }}
  </dl>
</div>

{{ if .Meta.Sections }}
<details class="sections" open>
  <summary>Sections ({{ len .Meta.Sections }})</summary>
  <ul>
    {{ range .Meta.Sections }}
    <li><a href="#L{{ .Line }}">L{{ .Line }}: {{ .Heading }}</a></li>
    {{ end }}
  </ul>
</details>
{{ end }}

{{ if .Meta.Versions }}
<details class="sections">
  <summary>Version history ({{ len .Meta.Versions }})</summary>
  <ul>
    {{ range .Meta.Versions }}
    <li>{{ .Date }} — <code>{{ .Hash }}</code>{{ if .Note }} — {{ .Note }}{{ end }}</li>
    {{ end }}
  </ul>
</details>
{{ end }}

{{ if and (isPDF .Meta.SourceType) .HasPages }}
<div class="side-by-side">
  <nav class="page-nav">
    <strong>Pages</strong>
    <ul>
      {{ range .Pages }}
      <li><a href="#L{{ .StartLine }}"{{ if $.Meta.URL }} title="View page {{ .Page }} in source PDF"{{ end }}>{{ if $.Meta.URL }}<a href="{{ $.Meta.URL }}#page={{ .Page }}" target="_blank">↗</a> {{ end }}Page {{ .Page }}</a></li>
      {{ end }}
    </ul>
  </nav>
  <div class="source-text">
  {{ range $i, $line := .Lines }}
  {{ if $line.PageBreak }}<div class="page-break" id="page{{ $line.PageNum }}"></div>{{ end }}
  <div class="source-line" id="L{{ add $i 1 }}"><span class="line-num"><a href="#L{{ add $i 1 }}"{{ if $line.SectionLink }} title="{{ $line.SectionHeading }}"{{ end }}>{{ add $i 1 }}</a></span><span class="line-text{{ if $line.IsHeading }} section-heading{{ end }}">{{ $line.Text }}</span></div>
  {{ end }}
  </div>
</div>
{{ else }}
<div class="source-text">
{{ range $i, $line := .Lines }}
<div class="source-line" id="L{{ add $i 1 }}"><span class="line-num"><a href="#L{{ add $i 1 }}"{{ if $line.SectionLink }} title="{{ $line.SectionHeading }}"{{ end }}>{{ add $i 1 }}</a></span><span class="line-text{{ if $line.IsHeading }} section-heading{{ end }}">{{ $line.Text }}</span></div>
{{ end }}
</div>
{{ end }}

</body>
</html>
`))

// Line is a single line of rendered output with metadata.
type Line struct {
	Text           string
	IsHeading      bool
	SectionLink    string // link to closest section in original source
	SectionHeading string
	PageBreak      bool // true if this line starts a new PDF page
	PageNum        int  // page number (if PageBreak is true)
}

// RenderData is the template context.
type RenderData struct {
	Meta        source.Meta
	HashChanged bool
	Lines       []Line
	HasPages    bool
	Pages       []source.PageBreak
	BasePath    string // relative path to docs root (e.g. ".." for refNN/ subdirs)
}

// Render writes an HTML reference document for the given source.
// basePath sets the relative path to the docs root for breadcrumb links
// (empty string for same directory, ".." for one level up).
func Render(w io.Writer, src *source.Source, basePath string) error {
	lines := buildLines(src)
	data := RenderData{
		Meta:        src.Meta,
		HashChanged: src.HasChanged(),
		Lines:       lines,
		HasPages:    len(src.Meta.Pages) > 0,
		Pages:       src.Meta.Pages,
		BasePath:    basePath,
	}
	return refTemplate.Execute(w, data)
}

// buildLines creates the line data, mapping each line to its closest section.
func buildLines(src *source.Source) []Line {
	textLines := strings.Split(src.Body, "\n")
	sectionLines := make(map[int]bool)
	for _, s := range src.Meta.Sections {
		sectionLines[s.Line] = true
	}

	// Build a set of page break lines for quick lookup.
	pageBreaks := make(map[int]int) // start_line → page number
	for _, p := range src.Meta.Pages {
		pageBreaks[p.StartLine] = p.Page
	}

	lines := make([]Line, len(textLines))
	for i, text := range textLines {
		lineNum := i + 1
		sec := src.SectionForLine(lineNum)

		l := Line{
			Text:      text,
			IsHeading: sectionLines[lineNum],
		}
		if sec != nil && sec.OriginalAnchor != "" && src.Meta.URL != "" {
			l.SectionLink = src.Meta.URL + sec.OriginalAnchor
			l.SectionHeading = sec.Heading
		}
		if pageNum, ok := pageBreaks[lineNum]; ok {
			l.PageBreak = true
			l.PageNum = pageNum
		}
		lines[i] = l
	}
	return lines
}

// RenderToString is a convenience wrapper that renders with default settings.
func RenderToString(src *source.Source) (string, error) {
	var buf strings.Builder
	if err := Render(&buf, src, ""); err != nil {
		return "", fmt.Errorf("rendering: %w", err)
	}
	return buf.String(), nil
}
