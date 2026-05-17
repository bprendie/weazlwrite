package tui

import "strings"

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
	m.helpView.SetContent(strings.TrimSpace(`
WeazlWrite help

Main writing
  tab                 Move between the editor/render pane and the tree.
  ctrl+e              Edit mode.
  ctrl+r              Render mode.
  ctrl+s              Save to the current target.
  ctrl+v              Save to: encrypted vault.
  alt+f               Save to: filesystem.
  ctrl+f              Find text in the current edit/render pane.
  ctrl+g              Jump to a page in the current edit/render pane.
  ctrl+n              New untitled vault note.
  ctrl+p              AI insert prompt. The generated Markdown block is inserted at the cursor.
  ctrl+o              Show or hide the tree.
  ctrl+y              Toggle mouse capture. Off means terminal drag-selection works for copying text.
  alt+o               Toggle eyes-only for the current vault note. Disabling asks for confirmation.
  ctrl+k              Open this command screen.
  ? or h              Open this command screen.
  ctrl+c              Quit.

Tree
  up/down or k/j      Move selection.
  enter               Open a file/note, or fold/unfold a folder.
  n                   New folder at the selected location.
  d                   Delete the selected file, note, or empty folder.
  r                   Rename or move the selected entry by typing its new path.
  o                   Toggle eyes-only on a vault note. Disabling asks for confirmation.
  space               Pick up the selected file/note; move to a destination folder; space again to drop.
  i                   Import the selected filesystem file/folder into the vault.
  pgup/pgdown         Page through the tree.
  esc                 Put down a picked-up entry and return focus to the writer.

Vault and filesystem
  Vault entries live inside the encrypted SQLite vault at ~/.weazlwrite/vault.
  Files entries are regular files from the current filesystem folder.
  Moving with space/drop stays inside the same side: vault-to-vault or filesystem-to-filesystem.
  To copy content between sides, use Save to: vault/filesystem or import.

Copying text
  Terminal mouse selection and app mouse scrolling compete for the same events.
  Press ctrl+y to turn mouse capture off, then select/copy text from edit or render with your terminal.
  Press ctrl+y again to restore mouse scrolling in the tree and panes.
  Eyes-only vault notes keep mouse capture on so blocks cannot be copied out with terminal selection.

Import
  Select a filesystem .md, .markdown, .txt, .pdf, or .docx file and press i to import it to the vault.
  Select a filesystem folder and press i to bulk-import it as a vault root.
  Folder imports preserve relative paths, skip hidden folders/files, and are meant for Obsidian-style vaults.
  PDF and DOCX imports are converted to pure Markdown before they are encrypted and saved.
  Image-only PDFs or image-only Word docs cannot be imported because there is no selectable text to convert.
  Existing vault paths are updated in place.

Prompts
  enter confirms.
  esc cancels.
`))
}
