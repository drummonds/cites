package render

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateThumbnails creates PNG thumbnails from a PDF using pdftoppm.
// Output files are named p01-1.png, p02-1.png, etc. in outDir.
// It renames them to p01.png, p02.png for consistency.
func GenerateThumbnails(pdfPath, outDir string, numPages int) error {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return fmt.Errorf("pdftoppm not found: %w", err)
	}

	for i := 1; i <= numPages; i++ {
		page := fmt.Sprintf("%d", i)
		prefix := filepath.Join(outDir, fmt.Sprintf("p%02d", i))
		cmd := exec.Command("pdftoppm",
			"-f", page, "-l", page,
			"-png", "-r", "150",
			pdfPath, prefix,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("pdftoppm page %d: %w\n%s", i, err, out)
		}
		// pdftoppm outputs prefix-N.png — rename to prefix.png.
		generated := fmt.Sprintf("%s-%d.png", prefix, i)
		target := prefix + ".png"
		if err := renameIfExists(generated, target); err != nil {
			// Try single-digit format: prefix-01.png
			generated = fmt.Sprintf("%s-%02d.png", prefix, i)
			_ = renameIfExists(generated, target)
		}
	}

	return nil
}

func renameIfExists(src, dst string) error {
	return os.Rename(src, dst)
}
