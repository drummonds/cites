// Package render generates paginated HTML reference documents from captured sources.
//
// RenderDir produces a directory with an index.html and per-page pNN.html files.
// For PDF sources, each page corresponds to a PDF page. For HTML/TXT, pages are
// chunked at ~100 lines.
package render

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/hum3/cites/internal/source"
)

// RenderOpts configures paginated rendering.
type RenderOpts struct {
	// PdfPath is the path to the original PDF for thumbnail generation.
	// If empty, thumbnails are skipped.
	PdfPath string

	// LinesPerPage is the number of lines per page for non-PDF sources.
	// Default: 100.
	LinesPerPage int

	// BasePath is the relative path back to the docs root (e.g. "..").
	BasePath string
}

// PageInfo describes one page of rendered output.
type PageInfo struct {
	PageNum   int    // 1-based page number
	File      string // e.g. "p01.html"
	StartLine int    // 1-based first line on this page
	EndLine   int    // 1-based last line on this page
	Thumbnail string // e.g. "p01.png" (empty if no thumbnail)
}

// PagesForSource splits the source into pages.
func PagesForSource(src *source.Source, linesPerPage int) []PageInfo {
	textLines := strings.Split(src.Body, "\n")
	totalLines := len(textLines)

	if linesPerPage <= 0 {
		linesPerPage = 100
	}

	var pages []PageInfo

	if src.Meta.SourceType == source.TypePDF && len(src.Meta.Pages) > 0 {
		// PDF: one page per PDF page boundary.
		for i, pb := range src.Meta.Pages {
			endLine := totalLines
			if i+1 < len(src.Meta.Pages) {
				endLine = src.Meta.Pages[i+1].StartLine - 1
			}
			pages = append(pages, PageInfo{
				PageNum:   pb.Page,
				File:      fmt.Sprintf("p%02d.html", pb.Page),
				StartLine: pb.StartLine,
				EndLine:   endLine,
			})
		}
	} else {
		// HTML/TXT: chunk by linesPerPage.
		pageNum := 1
		for start := 1; start <= totalLines; start += linesPerPage {
			end := start + linesPerPage - 1
			if end > totalLines {
				end = totalLines
			}
			pages = append(pages, PageInfo{
				PageNum:   pageNum,
				File:      fmt.Sprintf("p%02d.html", pageNum),
				StartLine: start,
				EndLine:   end,
			})
			pageNum++
		}
	}

	return pages
}

// IndexData is the template context for the index page.
type IndexData struct {
	Meta        source.Meta
	HashChanged bool
	TOC         []TOCEntry // combined pages + sections
	BasePath    string
	CurrentHash string // hash of the current version
}

// TOCEntry is either a page header or a section within a page.
type TOCEntry struct {
	IsPage  bool   // true = page row, false = section row
	PageNum int    // page number (for page rows)
	File    string // e.g. "p03.html"
	Heading string // section heading (for section rows)
	Link    string // e.g. "p03.html#l142"
	Lines   string // e.g. "1–45" (for page rows)
}

// PageData is the template context for a single page.
type PageData struct {
	Meta       source.Meta
	PageInfo   PageInfo
	Lines      []Line
	PrevPage   string // "" if first page
	NextPage   string // "" if last page
	TotalPages int
	BasePath   string
	HasThumb   bool
	ThumbFile  string // e.g. "p01.png"
}

// RenderDir generates a paginated reference site into outDir.
func RenderDir(outDir string, src *source.Source, opts RenderOpts) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	lpp := opts.LinesPerPage
	if lpp <= 0 {
		lpp = 100
	}

	pages := PagesForSource(src, lpp)
	allLines := buildLines(src)
	textLines := strings.Split(src.Body, "\n")

	// Generate thumbnails if PDF path provided.
	var thumbMap map[int]bool
	if opts.PdfPath != "" && src.Meta.SourceType == source.TypePDF {
		thumbMap = make(map[int]bool)
		if err := GenerateThumbnails(opts.PdfPath, outDir, len(pages)); err != nil {
			// Warn but continue without thumbnails.
			fmt.Fprintf(os.Stderr, "warning: thumbnail generation failed: %v\n", err)
		} else {
			for _, p := range pages {
				thumbFile := fmt.Sprintf("p%02d.png", p.PageNum)
				if _, err := os.Stat(filepath.Join(outDir, thumbFile)); err == nil {
					thumbMap[p.PageNum] = true
				}
			}
		}
	}

	// Render each page.
	for i, page := range pages {
		// Set thumbnail info on page.
		if thumbMap != nil && thumbMap[page.PageNum] {
			page.Thumbnail = fmt.Sprintf("p%02d.png", page.PageNum)
			pages[i] = page
		}

		// Extract lines for this page.
		startIdx := page.StartLine - 1
		endIdx := page.EndLine
		if endIdx > len(textLines) {
			endIdx = len(textLines)
		}
		if startIdx > len(allLines) {
			startIdx = len(allLines)
		}
		if endIdx > len(allLines) {
			endIdx = len(allLines)
		}
		pageLines := allLines[startIdx:endIdx]

		var prevPage, nextPage string
		if i > 0 {
			prevPage = pages[i-1].File
		}
		if i+1 < len(pages) {
			nextPage = pages[i+1].File
		}

		pd := PageData{
			Meta:       src.Meta,
			PageInfo:   page,
			Lines:      pageLines,
			PrevPage:   prevPage,
			NextPage:   nextPage,
			TotalPages: len(pages),
			BasePath:   opts.BasePath,
			HasThumb:   page.Thumbnail != "",
			ThumbFile:  page.Thumbnail,
		}

		outPath := filepath.Join(outDir, page.File)
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outPath, err)
		}
		if err := pageTemplate.Execute(f, pd); err != nil {
			f.Close()
			return fmt.Errorf("rendering %s: %w", page.File, err)
		}
		f.Close()
	}

	// Build combined TOC: interleave page headers with their sections.
	var toc []TOCEntry
	for _, page := range pages {
		toc = append(toc, TOCEntry{
			IsPage:  true,
			PageNum: page.PageNum,
			File:    page.File,
			Lines:   fmt.Sprintf("%d–%d", page.StartLine, page.EndLine),
		})
		for _, sec := range src.Meta.Sections {
			if sec.Line >= page.StartLine && sec.Line <= page.EndLine {
				toc = append(toc, TOCEntry{
					Heading: sec.Heading,
					Link:    fmt.Sprintf("%s#l%d", page.File, sec.Line),
				})
			}
		}
	}

	// Current version hash.
	currentHash := src.Meta.ContentHash

	// Render index.
	id := IndexData{
		Meta:        src.Meta,
		HashChanged: src.HasChanged(),
		TOC:         toc,
		BasePath:    opts.BasePath,
		CurrentHash: currentHash,
	}

	indexPath := filepath.Join(outDir, "index.html")
	f, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer f.Close()
	if err := indexTemplate.Execute(f, id); err != nil {
		return fmt.Errorf("rendering index: %w", err)
	}

	return nil
}

// Shared CSS for all templates.
const cssBlock = `
    :root { --bg: #fff; --fg: #1a1a1a; --muted: #666; --border: #ddd; --link: #0366d6; --line-bg: #f6f8fa; --topbar-bg: #0366d6; }
    @media (prefers-color-scheme: dark) {
      :root { --bg: #0d1117; --fg: #c9d1d9; --muted: #8b949e; --border: #30363d; --link: #58a6ff; --line-bg: #161b22; --topbar-bg: #1f6feb; }
    }
    * { box-sizing: border-box; }
    body { font-family: system-ui, sans-serif; max-width: 960px; margin: 2rem auto; padding: 0 1rem; background: var(--bg); color: var(--fg); }
    a { color: var(--link); }
    .topbar { background: var(--topbar-bg); color: #fff; padding: 0.5rem 1.5rem; margin-bottom: 1.5rem; border-radius: 6px; font-size: 0.85rem; display: flex; align-items: center; gap: 0.5rem; }
    .topbar a { color: #fff; text-decoration: none; font-weight: 600; }
    .topbar a:hover { text-decoration: underline; }
    .topbar .sep { opacity: 0.6; }
    .meta { border: 1px solid var(--border); border-radius: 6px; padding: 1rem 1.5rem; margin-bottom: 2rem; background: var(--line-bg); }
    .meta h1 { margin: 0 0 0.5rem; font-size: 1.4rem; }
    .meta dl { display: grid; grid-template-columns: auto 1fr; gap: 0.25rem 1rem; margin: 0; }
    .meta dt { font-weight: 600; color: var(--muted); }
    .meta dd { margin: 0; }
    .hash-ok { color: #2ea043; }
    .hash-changed { color: #d73a49; font-weight: bold; }
    .sections { margin-bottom: 2rem; }
    .sections summary { cursor: pointer; font-weight: 600; }
    .sections ul { list-style: none; padding-left: 1rem; }
    .source-text { font-family: 'SFMono-Regular', Consolas, monospace; font-size: 0.85rem; line-height: 1.6; }
    .source-line { display: flex; }
    .source-line:target { background: #fff8c5; }
    @media (prefers-color-scheme: dark) { .source-line:target { background: #3b2e00; } }
    .line-num { flex: 0 0 5ch; text-align: right; padding-right: 1ch; color: var(--muted); user-select: none; }
    .line-num a { color: inherit; text-decoration: none; }
    .line-num a:hover { color: var(--link); }
    .line-text { flex: 1; white-space: pre-wrap; word-break: break-word; }
    .section-heading { font-weight: bold; color: var(--link); }
    .page-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(6rem, 1fr)); gap: 0.5rem; margin: 1rem 0; }
    .page-list a { display: block; padding: 0.5rem; border: 1px solid var(--border); border-radius: 4px; text-align: center; text-decoration: none; }
    .page-list a:hover { background: var(--line-bg); }
    .nav-bar { display: flex; justify-content: space-between; align-items: center; margin: 1rem 0; padding: 0.5rem 0; border-top: 1px solid var(--border); }
    .nav-bar a { text-decoration: none; padding: 0.25rem 0.75rem; border: 1px solid var(--border); border-radius: 4px; }
    .nav-bar a:hover { background: var(--line-bg); }
    .nav-bar .disabled { color: var(--muted); border-color: var(--border); pointer-events: none; opacity: 0.4; }
    .side-layout { display: grid; grid-template-columns: 200px 1fr; gap: 1rem; }
    .thumb img { max-width: 100%; border: 1px solid var(--border); border-radius: 4px; }
    @media (max-width: 700px) { .side-layout { grid-template-columns: 1fr; } }
`

var indexFuncMap = template.FuncMap{
	"eq": func(a, b string) bool { return a == b },
}

var indexTemplate = template.Must(template.New("index").Funcs(indexFuncMap).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Meta.Title }} — Source Reference</title>
  <style>` + cssBlock + `
    .toc { border-collapse: collapse; width: 100%; margin: 1rem 0; }
    .toc th { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 2px solid var(--border); color: var(--muted); font-size: 0.85rem; }
    .toc td { padding: 0.25rem 0.75rem; border-bottom: 1px solid var(--border); }
    .toc .page-row { background: var(--line-bg); font-weight: 600; }
    .toc .page-row td { padding-top: 0.5rem; padding-bottom: 0.5rem; }
    .toc .section-row td { padding-left: 2rem; font-weight: normal; }
    .toc a { text-decoration: none; }
    .toc a:hover { text-decoration: underline; }
    .version-current { font-weight: 600; }
    .version-current::before { content: ""; }
  </style>
</head>
<body>

<nav class="topbar"><a href="{{ .BasePath }}/index.html">cites</a> <span class="sep">›</span> {{ .Meta.Title }}</nav>

<div class="meta">
  <h1>{{ .Meta.Title }}</h1>
  <dl>
    {{ if .Meta.URL }}<dt>Source</dt><dd><a href="{{ .Meta.URL }}">{{ .Meta.URL }}</a></dd>{{ end }}
    <dt>Type</dt><dd>{{ .Meta.SourceType }}</dd>
    {{ if .Meta.SourceDate }}<dt>Source date</dt><dd>{{ .Meta.SourceDate }}</dd>{{ end }}
    <dt>First captured</dt><dd>{{ .Meta.FirstCaptured.Format "2006-01-02 15:04 UTC" }}</dd>
    <dt>Last checked</dt><dd>{{ .Meta.LastChecked.Format "2006-01-02 15:04 UTC" }}</dd>
    <dt>Content hash</dt><dd><code{{ if .HashChanged }} class="hash-changed" title="Content has changed since capture"{{ else }} class="hash-ok"{{ end }}>{{ .Meta.ContentHash }}</code></dd>
    {{ if .HashChanged }}<dt>Status</dt><dd class="hash-changed">Content modified since capture</dd>{{ end }}
  </dl>
</div>

{{ if .TOC }}
<h2>Contents</h2>
<table class="toc">
  <thead><tr><th>Page</th><th>Section</th><th>Lines</th></tr></thead>
  <tbody>
  {{ range .TOC }}
  {{ if .IsPage }}
  <tr class="page-row"><td><a href="{{ .File }}">Page {{ .PageNum }}</a></td><td></td><td>{{ .Lines }}</td></tr>
  {{ else }}
  <tr class="section-row"><td></td><td><a href="{{ .Link }}">{{ .Heading }}</a></td><td></td></tr>
  {{ end }}
  {{ end }}
  </tbody>
</table>
{{ end }}

{{ if .Meta.Versions }}
<details class="sections" open>
  <summary>Version history ({{ len .Meta.Versions }})</summary>
  <ul>
    {{ range .Meta.Versions }}
    <li{{ if eq .Hash $.CurrentHash }} class="version-current"{{ end }}>{{ .Date }} — <code>{{ .Hash }}</code>{{ if .Note }} — {{ .Note }}{{ end }}{{ if eq .Hash $.CurrentHash }} (current){{ end }}</li>
    {{ end }}
  </ul>
</details>
{{ end }}

</body>
</html>
`))

var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
}

var pageTemplate = template.Must(template.New("page").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Meta.Title }} — Page {{ .PageInfo.PageNum }}</title>
  <style>` + cssBlock + `</style>
</head>
<body>

<nav class="topbar"><a href="{{ .BasePath }}/index.html">cites</a> <span class="sep">›</span> <a href="index.html">{{ .Meta.Title }}</a> <span class="sep">›</span> Page {{ .PageInfo.PageNum }}</nav>

{{ if .HasThumb }}
<div class="side-layout">
  <div class="thumb">
    <img src="{{ .ThumbFile }}" alt="Page {{ .PageInfo.PageNum }} thumbnail">
  </div>
  <div class="source-text">
  {{ range $i, $line := .Lines }}
  <div class="source-line" id="l{{ add $.PageInfo.StartLine $i }}"><span class="line-num"><a href="#l{{ add $.PageInfo.StartLine $i }}"{{ if $line.SectionLink }} title="{{ $line.SectionHeading }}"{{ end }}>{{ add $.PageInfo.StartLine $i }}</a></span><span class="line-text{{ if $line.IsHeading }} section-heading{{ end }}">{{ $line.Text }}</span></div>
  {{ end }}
  </div>
</div>
{{ else }}
<div class="source-text">
{{ range $i, $line := .Lines }}
<div class="source-line" id="l{{ add $.PageInfo.StartLine $i }}"><span class="line-num"><a href="#l{{ add $.PageInfo.StartLine $i }}"{{ if $line.SectionLink }} title="{{ $line.SectionHeading }}"{{ end }}>{{ add $.PageInfo.StartLine $i }}</a></span><span class="line-text{{ if $line.IsHeading }} section-heading{{ end }}">{{ $line.Text }}</span></div>
{{ end }}
</div>
{{ end }}

<div class="nav-bar">
  {{ if .PrevPage }}<a href="{{ .PrevPage }}">← Previous</a>{{ else }}<span class="disabled">← Previous</span>{{ end }}
  <span>Page {{ .PageInfo.PageNum }} of {{ .TotalPages }}</span>
  {{ if .NextPage }}<a href="{{ .NextPage }}">Next →</a>{{ else }}<span class="disabled">Next →</span>{{ end }}
</div>

</body>
</html>
`))
