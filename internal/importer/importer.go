package importer

import (
	"errors"
	"fmt"
	"github.com/ledongthuc/pdf"
	"os"
	"path/filepath"
	"strings"
)

var ErrImageOnlyDocument = errors.New("image-based documents cannot be imported; no selectable text was found")

type Document struct {
	Markdown string
	Warnings []string
}

func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".pdf", ".docx":
		return true
	default:
		return false
	}
}

func MarkdownPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + ".md"
	}
	return strings.TrimSuffix(path, ext) + ".md"
}

func Convert(path string) (Document, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		content, err := os.ReadFile(path)
		return Document{Markdown: string(content)}, err
	case ".txt":
		content, err := os.ReadFile(path)
		if err != nil {
			return Document{}, err
		}
		return Document{Markdown: textToMarkdown(path, string(content))}, nil
	case ".pdf":
		return pdfToMarkdown(path)
	case ".docx":
		return docxToMarkdown(path)
	case ".doc":
		return Document{}, fmt.Errorf("legacy .doc files are not supported; save as .docx first")
	default:
		return Document{}, fmt.Errorf("unsupported import type: %s", filepath.Ext(path))
	}
}

func textToMarkdown(path, text string) string {
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	body := strings.TrimSpace(normalizeText(text))
	if body == "" {
		return "# " + title + "\n"
	}
	return "# " + title + "\n\n" + body + "\n"
}

func pdfToMarkdown(path string) (Document, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer file.Close()

	var b strings.Builder
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")

	pagesWithText := 0
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			return Document{}, err
		}
		text = strings.TrimSpace(normalizeText(text))
		if text == "" {
			continue
		}
		pagesWithText++
		b.WriteString("## Page ")
		b.WriteString(fmt.Sprintf("%d", i))
		b.WriteString("\n\n")
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	if pagesWithText == 0 {
		return Document{}, ErrImageOnlyDocument
	}
	return Document{Markdown: strings.TrimSpace(b.String()) + "\n"}, nil
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = strings.Join(lines, "\n")
	text = blankLines.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}
