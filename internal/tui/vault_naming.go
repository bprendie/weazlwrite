package tui

import (
	"fmt"
	"html"
	"path"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func suggestedVaultName(content string) string {
	// Bound parsing work even for large pasted documents.
	if len(content) > 16384 {
		content = content[:16384]
	}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				lines = lines[i+1:]
				break
			}
		}
	}
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "# Untitled" {
		lines = lines[1:]
	}
	for i, line := range lines {
		prefix := parseMarkdownPrefix(line)
		if prefix.task != "" {
			lines[i] = prefix.indent + prefix.quote + prefix.marker + " " + strings.TrimPrefix(line, prefix.full)
		}
	}
	source := []byte(strings.Join(lines, "\n"))
	doc := goldmark.DefaultParser().Parse(text.NewReader(source))
	var visible strings.Builder
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			if n.Type() == ast.TypeBlock {
				visible.WriteByte(' ')
			}
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock, *ast.AutoLink:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			visible.Write(n.Value(source))
			if n.SoftLineBreak() || n.HardLineBreak() {
				visible.WriteByte(' ')
			}
		case *ast.String:
			visible.Write(n.Value)
		}
		return ast.WalkContinue, nil
	})
	words := strings.FieldsFunc(html.UnescapeString(visible.String()), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsMark(r)
	})
	if len(words) > 6 {
		words = words[:6]
	}
	name := []rune(strings.ToLower(strings.Join(words, "-")))
	if len(name) > 60 {
		name = name[:60]
	}
	stem := strings.Trim(string(name), "-")
	if stem == "" {
		stem = "untitled"
	}
	return stem + ".md"
}

func (m model) draftNameSuggestion() string {
	base := vaultParent(m.vaultPath)
	if base != "" {
		base += "/"
	}
	proposed := base + suggestedVaultName(m.editorText())
	notes, err := m.store.ListNotes()
	if err != nil {
		return proposed
	}
	used := map[string]bool{}
	for _, note := range notes {
		if note.ID != m.vaultID {
			used[note.Path] = true
		}
	}
	candidate := proposed
	for i := 2; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s-%d.md", strings.TrimSuffix(proposed, ".md"), i)
	}
	return candidate
}

func (m model) firstVaultSave(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dirty || m.saves.inFlight != nil {
		m.saves.pending = key
		cmd := m.startVaultSave(true)
		return m, cmd
	}
	return m.startSaveVault()
}

func (m *model) acceptDraftName(name string) error {
	name = cleanVaultDocumentPath(name)
	if name == "" {
		return fmt.Errorf("invalid vault filename")
	}
	chosen, err := m.store.SaveDraft(m.vaultID, name, titleFor(name, m.editorText()), m.editorText(), false)
	if err != nil {
		return err
	}
	m.vaultPath, m.filePath = chosen, chosen
	m.autoNamed, m.dirty = false, false
	m.saves.confirm = false
	m.saves.err, m.err = "", ""
	m.status = "saved vault:" + chosen
	m.expandTreeTo("vault:" + chosen)
	return nil
}

func autoDraftPath(current, content string) string {
	return path.Join(path.Dir(current), suggestedVaultName(content))
}
