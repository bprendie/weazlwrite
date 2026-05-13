package importer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
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

func docxToMarkdown(path string) (Document, error) {
	rc, err := openDocxDocument(path)
	if err != nil {
		return Document{}, err
	}
	defer rc.Close()

	paragraphs, err := parseDocxParagraphs(rc)
	if err != nil {
		return Document{}, err
	}
	if len(paragraphs) == 0 {
		return Document{}, ErrImageOnlyDocument
	}

	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, p := range paragraphs {
		text := strings.TrimSpace(normalizeText(p.text))
		if text == "" {
			continue
		}
		switch {
		case strings.HasPrefix(strings.ToLower(p.style), "heading1"):
			b.WriteString("# ")
		case strings.HasPrefix(strings.ToLower(p.style), "heading2"):
			b.WriteString("## ")
		case strings.HasPrefix(strings.ToLower(p.style), "heading3"):
			b.WriteString("### ")
		}
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	out := strings.TrimSpace(b.String())
	if out == "# "+title {
		return Document{}, ErrImageOnlyDocument
	}
	return Document{Markdown: out + "\n"}, nil
}

func openDocxDocument(path string) (io.ReadCloser, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	for _, file := range zr.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			zr.Close()
			return nil, err
		}
		return closeBoth{ReadCloser: rc, close: zr.Close}, nil
	}
	zr.Close()
	return nil, fmt.Errorf("word/document.xml not found")
}

type closeBoth struct {
	io.ReadCloser
	close func() error
}

func (c closeBoth) Close() error {
	err := c.ReadCloser.Close()
	if closeErr := c.close(); err == nil {
		err = closeErr
	}
	return err
}

type docxParagraph struct {
	text  string
	style string
}

func parseDocxParagraphs(r io.Reader) ([]docxParagraph, error) {
	decoder := xml.NewDecoder(r)
	var paragraphs []docxParagraph
	var current *docxParagraph
	inText := false

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "p":
				current = &docxParagraph{}
			case "pStyle":
				if current != nil {
					current.style = attrValue(token.Attr, "val")
				}
			case "t":
				if current != nil {
					inText = true
				}
			case "tab":
				if current != nil {
					current.text += "\t"
				}
			case "br", "cr":
				if current != nil {
					current.text += "\n"
				}
			}
		case xml.CharData:
			if current != nil && inText {
				current.text += string([]byte(token))
			}
		case xml.EndElement:
			switch token.Name.Local {
			case "t":
				inText = false
			case "p":
				if current != nil && strings.TrimSpace(current.text) != "" {
					paragraphs = append(paragraphs, *current)
				}
				current = nil
				inText = false
			}
		}
	}
	return paragraphs, nil
}

func attrValue(attrs []xml.Attr, local string) string {
	for _, attr := range attrs {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

var blankLines = regexp.MustCompile(`\n{3,}`)

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

func ConvertBytesDocx(content []byte) (Document, error) {
	return docxToMarkdownReader(bytes.NewReader(content), int64(len(content)), "document.docx")
}

func docxToMarkdownReader(r io.ReaderAt, size int64, name string) (Document, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Document{}, err
	}
	for _, file := range zr.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return Document{}, err
		}
		defer rc.Close()
		paragraphs, err := parseDocxParagraphs(rc)
		if err != nil {
			return Document{}, err
		}
		if len(paragraphs) == 0 {
			return Document{}, ErrImageOnlyDocument
		}
		title := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		var b strings.Builder
		b.WriteString("# ")
		b.WriteString(title)
		b.WriteString("\n\n")
		for _, p := range paragraphs {
			text := strings.TrimSpace(normalizeText(p.text))
			if text != "" {
				b.WriteString(text)
				b.WriteString("\n\n")
			}
		}
		return Document{Markdown: strings.TrimSpace(b.String()) + "\n"}, nil
	}
	return Document{}, fmt.Errorf("word/document.xml not found")
}
