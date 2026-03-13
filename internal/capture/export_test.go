package capture

// Export private functions for testing.
// The capture package uses this pattern to test internal functions
// from external test files if needed, but since our tests are in
// the same package, these are available directly.
//
// This file exists to document the exported API surface:
//   - Extract (public)
//   - Capture (public)
//   - CapturePDFPages (public)
//   - detectPDFPages (internal, tested in capture_test.go)
//   - extractHTML (internal, tested in capture_test.go)
//   - detectSections (internal, tested in capture_test.go)
//   - slugify (internal, tested in capture_test.go)
//   - canonicalURL (internal, tested in capture_test.go)
