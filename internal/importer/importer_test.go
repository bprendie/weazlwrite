package importer

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownPathNormalizesImportedDocumentNames(t *testing.T) {
	if got := MarkdownPath("notes/spec.pdf"); got != "notes/spec.md" {
		t.Fatalf("MarkdownPath(pdf) = %q", got)
	}
	if got := MarkdownPath("notes/spec.docx"); got != "notes/spec.md" {
		t.Fatalf("MarkdownPath(docx) = %q", got)
	}
}

func TestConvertDocxExtractsTextAsMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.docx")
	if err := writeDocx(path, `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Overview</w:t></w:r>
    </w:p>
    <w:p><w:r><w:t>Hello from Word.</w:t></w:r></w:p>
  </w:body>
</w:document>`); err != nil {
		t.Fatal(err)
	}

	doc, err := Convert(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Markdown, "## Overview") {
		t.Fatalf("converted markdown missing heading: %q", doc.Markdown)
	}
	if !strings.Contains(doc.Markdown, "Hello from Word.") {
		t.Fatalf("converted markdown missing paragraph: %q", doc.Markdown)
	}
}

func writeDocx(path, documentXML string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	doc, err := zw.Create("word/document.xml")
	if err != nil {
		zw.Close()
		return err
	}
	if _, err := doc.Write([]byte(documentXML)); err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}
