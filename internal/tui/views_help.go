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
		line(keyCycleFocus, "Indent or nest a list item while editing; otherwise switch panes."),
		line("enter", "Continue Markdown lists/quotes; an empty item steps out."),
		line("backspace", "At the start of list text, unnest or remove the marker."),
		line("shift+tab", "Outdent the current line (up to four spaces or one tab)."),
		line(keyEdit, "Edit mode from the tree or preview. In the editor, end of line."),
		line(keyRender, "Render mode."),
		line(keySave, "Save now; vault notes also save while typing."),
		line(keySaveVault, "Save to: encrypted vault."),
		line(keySaveDisk, "Save to: filesystem."),
		line(keyFind, "Find text; editor search stays below the document."),
		line("f3 / shift+f3", "Next / previous editor match (alt+f3 also goes backward)."),
		line("search: tab", "Switch between find and replacement; Enter replaces one match."),
		line("search: f2 / esc", "Edit the selected match / close search and restore position."),
		line("alt+w / alt+t", "Cycle prose width / toggle typewriter scrolling."),
		line(keyJumpPage, "Jump to a page in the current edit/render pane."),
		line(keyNewNote, "New untitled vault note."),
		line(keyAI, "AI insert prompt. The generated Markdown block is inserted at the cursor."),
		line(keyToggleTree, "Show or hide the tree."),
		line("shift+arrows", "Select text; ctrl+shift+left/right selects by word."),
		line("ctrl+a", "Select all editor text. Home moves to line start."),
		line("ctrl+c / ctrl+x", "Copy / cut editor selection; Eyes Only blocks clipboard export."),
		line(keyUndo, "Undo the last edit."),
		line("ctrl+y / alt+z", "Redo (ctrl+shift+z also works where supported)."),
		line(keyRedo, "Redo the last undone edit."),
		line(keySelection, "Toggle app mouse capture. Off lets the terminal drag-select text."),
		line(keyEyes, "Toggle eyes-only for the current vault note. Disabling asks for confirmation."),
		line(keyHelp, "Open this command screen."),
		line(keyHelpQuery+" or "+keyHelpH, "Open this command screen when not typing in the editor."),
		line("ctrl+q", "Quit after vault saves; unsaved disk drafts ask save/discard/cancel."),
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
		line("edit mode", "Clear text selection first; otherwise reveal/focus the tree and flush vault edits."),
		"",
		"Vault and filesystem",
		"  Vault entries live inside the encrypted SQLite vault at ~/.weazlwrite/vault.",
		"  Files entries are regular files from the current filesystem folder.",
		"  Moving with space/drop stays inside the same side: vault-to-vault or filesystem-to-filesystem.",
		"  To copy content between sides, use Save to: vault/filesystem or import.",
		"",
		"Copying text",
		"  Terminal mouse selection and app mouse scrolling compete for the same events.",
		"  Drag selects a range for editing. Ctrl+C copies, Ctrl+X cuts; click only places the cursor.",
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
