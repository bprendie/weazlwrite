package importer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

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
