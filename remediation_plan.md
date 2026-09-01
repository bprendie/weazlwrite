# WeazlWrite Editor Remediation Plan

Baseline: `e9b7e3c` Checkpoint before first-class editor wrap work.

This plan makes the writing surface first-class: a real keymap, soft word wrap that follows the pane, and paging/find/selection that follow what you see. It does not change vault, import, or LLM behavior.

Nothing below is decided except the direction. Phase 1 (keymap) is the discussion we should have before any wrap work, because wrap navigation is useless if the chords still belong to a form widget.

## Required after every change

Do not leave a phase or patch at tests-only. After any code change:

1. Build both repo binaries.
2. Install them over the copies on `PATH`.

```sh
go test ./...
go build -o weazlwrite ./cmd/weazlwrite
go build -o weazlwrite-setup ./cmd/weazlwrite-setup
install -m 755 weazlwrite "$HOME/.weazlwrite/bin/weazlwrite"
install -m 755 weazlwrite-setup "$HOME/.weazlwrite/bin/weazlwrite-setup"
```

The installed app is `$HOME/.weazlwrite/bin/weazlwrite`, which is what the shell runs. A repo-local `weazlwrite` that is not copied there does not count as done. If a change only touches `cmd/weazlwrite` / `internal/tui`, still build and install `weazlwrite`; install `weazlwrite-setup` as well whenever setup code changed, and always when doing a full phase close-out.

## Current Shape

Edit mode is Charmbracelet `textarea` in `internal/tui`. Resize sets width/height to the main panel. App keys are intercepted in `updateWrite` *before* focus is considered, then leftovers go to `editor.Update`.

Render mode is a `viewport` plus Glamour, which already word-wraps.

Selection mode (`ctrl+y`) splits on `\n` and truncates.

So there are three wrap models, and they do not share coordinates:

| Surface | Wrap | Line unit |
|---|---|---|
| Editor | Soft wrap inside `textarea` | Logical (`\n`) for paging/find/status |
| Preview | Glamour `WithWordWrap` | Rendered rows |
| Selection | None (`ansi.Truncate`) | Logical rows |

`textarea` already soft-wraps on Unicode space and moves the cursor on visual rows. `Value()` keeps the author's newlines. That storage model is correct for Markdown. The widget is still configured as a short form:

- `MaxHeight` defaults to 99, and Enter is refused once the buffer has 99 logical lines.
- `MaxWidth` defaults to 500, so wrap can stop following a wide pane.
- Library hard cap is 10,000 logical lines.
- Default prompt `┃ ` plus a 4-column line-number gutter plus the panel border all eat wrap width.
- `styles.editor` is unused.
- Tabs are expanded to four spaces on insert *and* on `SetValue`, so opening a tab-indented file can rewrite it on save.

## Goals

1. While writing, motion and edit keys belong to the buffer.
2. Long paragraphs wrap to the pane in edit mode. Cursor, Home/End, and page keys follow visual rows.
3. Status page counts, `ctrl+g`, find, and selection use the same visual-row idea as the editor.
4. Mouse in the writing pane is a strong preference: click places the cursor, drag selects, wheel already scrolls. Keyboard remains complete for SSH and 80x24.
5. Preview wrap stays as it is (Glamour). Do not make the editor Markdown-aware in the first pass.
6. Files keep the author's hard newlines. Soft wrap is visual only.

Non-goals for this pass: undo/redo, Markdown hanging indent, a preferred wrap column, hard-reflow-on-save, forking `textarea`.

## Phase 1 — Keymap ownership

**Why first:** wrap and paging will add Home/End, visual-line motion, and later paste/indent. Those collide with the current global intercept list. Remap once, then build wrap on a keymap that will survive.

**Problem:** `updateWrite` is a global switch. Several app chords steal `textarea` defaults even when the writer is focused.

| Chord | App today | `textarea` default | While writing |
|---|---|---|---|
| `ctrl+v` | save to vault | paste | paste is dead |
| `alt+f` | save to filesystem | word forward | word forward is dead |
| `ctrl+p` | AI insert | previous line | extra up-line chord is dead |
| `ctrl+n` | new vault note (editor) / new folder (tree) | next line | extra down-line chord is dead |
| `ctrl+k` | help | kill to end of line | kill-line is dead |
| `ctrl+e` | edit mode | end of line | end-of-line is dead (`ctrl+a` still works) |
| `ctrl+f` | find | forward char | keep as find (writer expectation) |
| `tab` | cycle tree/editor | not bound | indent is dead |
| `ctrl+y` | selection mode | unused | later undo/redo conflict |

Arrows, Backspace, Delete, Enter, `ctrl+a`, `ctrl+u`, `ctrl+w`, and Home/End in the editor already reach `textarea`. Home/End are logical-line, not visual-line; that is a wrap bug, not a keymap bug.

### Proposed rule

Split the keymap by focus.

- **Editor-focused:** buffer owns motion and edit. App commands that must work while typing use chords that do not collide with paste, line start/end, word motion, or kill-line.
- **Tree-focused:** letter keys stay as they are (`n`, `d`, `r`, `o`, `i`, `space`, `j`/`k`). App chords that are harmless in the tree may keep their current meaning.
- **Preview-focused:** pane owns scroll. `esc` already returns to edit.

Do not globally intercept a chord that the editor needs. Intercept it only when the editor is not focused, or move the app command.

### Proposed chords

Keep (no collision, or collision we accept):

| Chord | Meaning |
|---|---|
| `ctrl+s` | save current target |
| `ctrl+f` | find |
| `ctrl+g` | jump to page |
| `ctrl+o` | toggle tree |
| `ctrl+r` | render mode |
| `ctrl+l` | LLM config |
| `ctrl+c` | quit |
| `alt+o` | Eyes Only on current vault note |
| `alt+i` | AI insert |
| `tab` | cycle tree / writing pane |
| `f1` | help |
| `?` | help when not typing in the editor |
| `pgup` / `pgdown` | page the focused pane |
| `esc` | selection off, else render→edit, else editor→tree |

Move:

| Today | Proposed | Restore to editor |
|---|---|---|
| `ctrl+v` save vault | `alt+v` | paste |
| `alt+f` save filesystem | `alt+d` | word forward |
| `ctrl+p` AI | drop; `alt+i` only | previous line |
| `ctrl+n` new note while writing | `alt+n` | next line |
| `ctrl+n` new folder | keep, **tree-focused only** | — |
| `ctrl+k` help | drop; `f1` / `?` | kill to end of line |
| `ctrl+e` edit mode | keep for tree/preview only; in editor it is line end | end of line |

`ctrl+y` stays selection mode for now. Undo is a later phase; do not spend `ctrl+z` / `ctrl+y` until that lands.

Tab stays pane focus in this pass. Indent is a later decision (`ctrl+]` or editor-owned Tab with `ctrl+tab` for panes).

### Implementation

- Extract a keymap table (`internal/tui/keymap.go`) used by `updateWrite`, help text, and the footer. Help and README must not be a second source of truth.
- Route by focus: editor-owned keys fall through to `editor.Update` when `focusEditor && viewEdit`.
- Customize `editor.KeyMap` so leftover emacs chords we do not want are unbound, rather than firing by surprise.
- Update `views_help.go`, `views_status.go` footer, `README.md` Keys, and tests.

Tests to add or retarget:

- `ctrl+v` in the editor does not open save-vault; `alt+v` does.
- `alt+d` opens save-filesystem.
- `ctrl+e` in the editor does not toggle view; from tree/preview it still enters edit.
- `ctrl+p` in the editor does not open AI; `alt+i` does.
- `ctrl+n` in the editor does not create a note; `alt+n` does. Tree `ctrl+n` still creates a folder.
- `tab` still cycles panes.
- `h` still types in the editor and still opens help from the tree.

### Discuss before coding

See Open Questions. The table above is a proposal, not a lock.

## Phase 2 — Make `textarea` a document widget

Uncap and restyle. No wrap-coordinate work yet.

- Set `MaxHeight = 0` and `MaxWidth = 0` (or large explicit values if `0` mis-sizes line numbers).
- Confirm Enter works past 99 logical lines, including after opening a long file.
- Drop or neutralize the default `┃ ` prompt so wrap width is the pane, not the pane minus a fake gutter.
- Keep line numbers, but give them a stable width so notes past line 99 do not overflow the reserved 4 columns.
- Apply `styles.editor` to focused/blurred textarea styles so the cursor line matches the rest of the TUI.
- Decide tab policy: preserve tabs in `SetValue`, or keep expand-to-spaces but do it only for typed input, not file load. Loading must not mutate a file on first save.
- `resize()` must keep using the focused panel frame size (thick vs normal border) so wrap width matches what is drawn.

Tests:

- Enter on a 120-line buffer inserts a newline.
- A 200-column pane with the tree hidden wraps near pane width, not at 500.
- Opening a file that contains tabs does not rewrite those tabs until the user edits that line (or whatever policy we pick).
- Existing “view does not exceed terminal width” tests still pass with a long unwrapped paragraph.

## Phase 3 — Visual-line coordinates and paging

Today `search_nav.go` pages with `editor.Line()` / `LineCount()`, which are hard newlines. Status `page N/M` and `ctrl+g` disagree with what is on screen, and disagree with preview.

- Add one helper that, given document text and wrap width, returns visual row count and can map (logical line, column) ↔ visual row. Match `textarea` wrap behavior as closely as possible, including the line-number/prompt gutter.
- Editor `pgup` / `pgdown` move by visual pages, not logical lines.
- `currentPage` / `totalPages` / `ctrl+g` use visual rows in edit mode. Preview keeps using the viewport line count.
- Home/End in the editor move to the start/end of the **visual** row. `ctrl+home` / `ctrl+end` (or preview-style Home/End when the preview is focused) stay document top/bottom.
- Up/down can keep using `textarea`’s built-in visual-row motion.

If the helper cannot match `textarea` wrap without copying private functions, stop and decide whether to wrap the widget or replace it. Do not ship two different wrap algorithms.

Tests: a single long paragraph with no `\n` is more than one page in a short pane; jump-to-page 2 moves the cursor; Home on a wrapped row does not jump to the start of the paragraph.

## Phase 4 — Find and selection follow wrap

Find:

- Still search the raw document, not the wrapped display.
- Position the cursor on the match (already does `SetCursor`).
- After Phase 3, paging/status around the match should be correct.
- Add find-next from the current column, not only from the next logical line. Find-previous can wait unless it is cheap.

Selection:

- Stop truncating. Render wrapped lines in selection view.
- Keep whole-line copy for this pass if character ranges are too much, but the *display* must wrap.
- Drive `selectOffset` from visual scroll, not `editor.Line()`.
- Eyes Only behavior stays: no copy.

Fold mouse selection into the live editor once wrap coordinates exist. Do not keep a second truncated selection view as the long-term model.

## Phase 5 — Mouse editing and paste

Bubble Tea's `textarea` does not map clicks to a cursor. WeazlWrite already captures cell-motion (`tea.WithMouseCellMotion`) and already maps pane bounds (`mainContentBounds`). Wheel scroll and pane-focus-on-click work. What is missing is hit-testing a cell onto `(logical line, column)` after wrap.

History: app mouse capture and terminal drag-select cannot both own the same events. `ctrl+y` first disabled capture so the terminal could copy; then in-app selection (`2745c66`) took over and never calls `DisableMouse`. Help text still talks about toggling capture off. Eyes Only must keep capture on so the terminal cannot harvest the buffer.

Once Phase 3 can map a visual cell:

- Click in the writing pane places the cursor. Click on the tree still focuses/selects the tree.
- Drag in the writing pane selects a character range on wrapped rows; release copies via the existing OSC-52 path.
- Click without drag does not copy.
- Eyes Only: click-to-cursor stays; copy on release is a no-op.
- Wheel stays as it is.
- Keyboard remains the full editor. Mouse is additive.
- If hit-testing is wrong on a real terminal (tmux, SSH, odd padding), ship click-to-cursor only and keep the current selection mode as fallback. Do not guess.

`ctrl+y` after this: either a true capture toggle for people who still want terminal select on non-Eyes-Only notes, or drop the mode if live drag-select is good enough. Decide after it works, not before.

Also in this phase: paste at cursor (`ctrl+v` once Phase 1 frees it), and `ctrl+left` / `ctrl+right` if the terminal delivers them. Indent/Tab remains a separate decision.

## Phase 6 — Optional, later

- Undo/redo. Revisit `ctrl+y` (selection) vs `ctrl+z` / `ctrl+shift+z`.
- Preferred wrap column (e.g. 80) with a right margin, vs always wrapping to the pane.
- Markdown-aware wrap: hanging list/quote indents, leave fenced code and tables alone.
- Hard wrap / reflow command that inserts real newlines. Default remains soft wrap.
- Replace or fork `textarea` if visual-line mapping, undo, or click-to-cursor cannot be done honestly on top of it.
- Performance: `textarea.View` rebuilds the whole wrapped document every frame.

## Suggested order of work

```
Phase 1 keymap  →  Phase 2 uncap widget  →  Phase 3 visual paging
                                              ↓
                         Phase 4 find/selection wrap   Phase 5 click-to-cursor + drag-select
```

Phase 5 depends on Phase 3 coordinates. It can start as soon as that mapping is honest; it should not wait on polish.

Do not start Phase 3 until Phase 1 is agreed. Do not start Phase 6 until someone is hitting a real limit.

## Files that will move

| Phase | Likely files |
|---|---|
| 1 | `internal/tui/input.go`, new `keymap.go`, `views_help.go`, `views_status.go`, `render_test.go`, `README.md` |
| 2 | `internal/tui/model.go`, `layout_state.go`, `styles.go`, tests |
| 3 | `internal/tui/search_nav.go`, new wrap helper, `input.go` Home/End |
| 4 | `internal/tui/search_nav.go`, `selection.go` |
| 5 | `internal/tui/input.go`, `selection.go`, wrap hit-test helper |

## Open questions

1. **Save destinations.** `alt+v` vault and `alt+d` disk are symmetric with `alt+o` / `alt+i` / `alt+n`. Alternatives: `ctrl+shift+s` as a save-as chooser, or a small save menu from `ctrl+s` when there is no current target. `ctrl+s` should stay “save here.”
2. **Find vs emacs `ctrl+f`.** Recommendation: keep find. Forward-char is already `right`.
3. **Tab.** Keep pane cycle for now, or make Tab indent when the editor is focused and move pane cycle to `ctrl+tab`? `ctrl+tab` is often eaten by the terminal/tmux.
4. **Help.** Dropping `ctrl+k` makes `f1` and `?` the documented help keys. Is `ctrl+k` as command palette (current help screen) worth keeping under another chord, e.g. `ctrl+shift+k`?
5. **New note while writing.** Tree `n` already creates a document. Is `alt+n` needed, or is “leave the editor, press `n`” enough?
6. **Wrap to pane vs wrap column.** Recommendation: pane width in this pass. A config column is Phase 6.
7. **Stay on `textarea`?** Phase 2–4 try to keep it. If visual wrap mapping cannot match the widget, replacing it becomes the plan, not a side quest.
8. **`ctrl+y` after live drag-select.** Keep a capture-off escape hatch for terminal copy on normal notes, or drop the mode. Eyes Only never disables capture.

## Manual smoke after each phase

Build and install first (see Required after every change). Smoke the installed binary, not only `go run`.

- Type a 200-line note; Enter still works.
- A long prose paragraph wraps in edit mode; cursor up/down stays in the paragraph.
- Hide the tree; wrap width grows.
- Save vault / save disk / save current still work from the new chords.
- AI insert, find, jump page, Eyes Only, selection copy, help, tree `n`/`ctrl+n`.
- `h` types in the editor.
- Preview wrap still looks like Markdown, not like the editor.
- A tab-indented file does not silently reindent on open+save.
- Click in the editor places the cursor; drag copies (OSC-52); Eyes Only click works and copy does not.
