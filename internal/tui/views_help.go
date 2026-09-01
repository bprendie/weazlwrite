package tui

import (
	"fmt"
	"strings"
)

func (m model) helpScreenView() string {
	innerH := m.bodyHeight()
	m.helpView.Height = contentHeight(m.styles.panel, innerH)
	m.helpView.Width = contentWidth(m.styles.panel, m.width)
	return renderPanel(m.styles.activePanel, m.width, innerH, m.helpView.View())
}

func (m *model) renderHelp() {
	if m.helpView.Width <= 0 {
		m.helpView.Width = max(20, contentWidth(m.styles.panel, m.width))
	}
	m.helpView.SetContent(helpContent())
}

func helpContent() string {
	line := func(chord, text string) string {
		return fmt.Sprintf("  %-20s%s", chord, text)
	}
	return strings.TrimSpace(strings.Join([]string{
		"WeazlWrite help",
		"",
		"Main writing",
		line(keyCycleFocus, "Move between the editor/render pane and the tree."),
		line(keyEdit, "Edit mode from the tree or preview. In the editor, end of line."),
		line(keyRender, "Render mode."),
		line(keySave, "Save to the current target."),
		line(keySaveVault, "Save to: encrypted vault."),
		line(keySaveDisk, "Save to: filesystem."),
		line(keyFind, "Find text in the current edit/render pane."),
		line(keyJumpPage, "Jump to a page in the current edit/render pane."),
		line(keyNewNote, "New untitled vault note."),
		line(keyAI, "AI insert prompt. The generated Markdown block is inserted at the cursor."),
		line(keyToggleTree, "Show or hide the tree."),
		line(keySelection, "Toggle app mouse capture. Off lets the terminal drag-select text."),
		line(keyEyes, "Toggle eyes-only for the current vault note. Disabling asks for confirmation."),
		line(keyHelp, "Open this command screen."),
		line(keyHelpQuery+" or "+keyHelpH, "Open this command screen when not typing in the editor."),
		line("ctrl+c", "Quit."),
		"",
		"Tree",
		line("up/down or k/j", "Move selection."),
		line("enter", "Open a file/note, or fold/unfold a folder."),
		line("n", "New document at the selected folder or file's parent; enter the path before creation."),
		line(keyNewFolder, "New folder at the selected location."),
		line("d", "Delete the selected file, note, or empty folder."),
		line("r", "Rename or move the selected entry by typing its new path."),
		line("o", "Toggle eyes-only on a vault note. Disabling asks for confirmation."),
		line("space", "Pick up the selected file/note; move to a destination folder; space again to drop."),
		line("i", "Import the selected filesystem file/folder into the vault."),
		line("pgup/pgdown", "Page through the tree."),
		line("esc", "Put down a picked-up entry and return focus to the writer."),
		"",
		"Escape",
		line("render mode", "Return to edit mode."),
		line("edit mode", "Move focus back to the tree when it is visible."),
		"",
		"Vault and filesystem",
		"  Vault entries live inside the encrypted SQLite vault at ~/.weazlwrite/vault.",
		"  Files entries are regular files from the current filesystem folder.",
		"  Moving with space/drop stays inside the same side: vault-to-vault or filesystem-to-filesystem.",
		"  To copy content between sides, use Save to: vault/filesystem or import.",
		"",
		"Copying text",
		"  Terminal mouse selection and app mouse scrolling compete for the same events.",
		"  Drag in the writing pane to copy a character range. Click without drag only moves the cursor.",
		"  Press " + keySelection + " to turn app mouse capture off so the terminal can drag-select.",
		"  Eyes-only vault notes keep mouse capture on so blocks cannot be copied out with terminal selection.",
		"",
		"Import",
		"  Select a filesystem .md, .markdown, .txt, .pdf, or .docx file and press i to import it to the vault.",
		"  Select a filesystem folder and press i to bulk-import it as a vault root.",
		"  Folder imports preserve relative paths, skip hidden folders/files, and are meant for Obsidian-style vaults.",
		"  PDF and DOCX imports are converted to pure Markdown before they are encrypted and saved.",
		"  Image-only PDFs or image-only Word docs cannot be imported because there is no selectable text to convert.",
		"  Existing vault paths are updated in place.",
		"",
		"Prompts",
		"  enter confirms.",
		"  esc cancels.",
	}, "\n"))
}
