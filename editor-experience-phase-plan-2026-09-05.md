# Editor Experience Phase Plan

Date: September 5, 2026
Status: Planning only; uncommitted. No application changes authorized by this document.

## Scope and agreed direction

Improve composing and revising documents in edit mode. Preserve the rest of the application's behavior and appearance except where editor focus and draft persistence require integration. Stay within Bubble Tea; assess whether to extend the existing Bubbles textarea or introduce a dedicated editor component when selection work begins.

Agreed requirements:

- Traditional editing keys take priority while writing. Tab belongs to indentation, not pane switching.
- Understand bullet and numbered lists, including continuation and nesting.
- Vault documents save continuously as the user types. Ctrl+S immediately flushes pending edits and confirms successful saving.
- Esc provides keyboard-only departure from the editor without introducing Vim normal/insert modes.
- Preserve undo across saves and protect pending writing when changing documents or leaving the editor.

Earlier review recommendations are included as later phases; detailed shortcuts and comfort defaults below are implementation proposals, not separately approved preferences. This plan governs this editor initiative; older plans describe their own historical scope.

## Phase 1 — Reliable editor input and keyboard ownership

- Route clipboard results and editor lifecycle messages back to the focused editor. Ensure successful paste participates in dirty tracking, undo, and subsequent rolling saves.
- Separate app shortcuts from editor shortcuts. Audit collisions, especially Tab, Ctrl+C, Ctrl+Y, and Alt+D; choose and document a coherent map before implementing selection. Provide terminal-compatible alternatives for modified keys that are not reliably distinguishable.
- Make Tab insert indentation and Shift+Tab remove indentation. Preserve existing literal tabs and hard newlines; define a consistent indentation width without rewriting an entire document.
- Make undo groups follow editing intent: end a group on cursor movement, selection changes, paste, and transitions between insertion and deletion. Treat paste as its own undoable action.
- Stop synchronously rendering the hidden Markdown preview on each edit. Refresh when entering preview; defer other expensive work as needed.

Acceptance:

- Clipboard paste and terminal bracketed paste both insert at the cursor and undo correctly, including multiline and Unicode content.
- Tab never moves focus away from a focused editor, even when the tree is visible.
- Moving within a line before editing creates a separate undo step.
- Existing wrapping, paging, tab preservation, and document loading behavior remain correct.
- Measure typing responsiveness with a substantial Markdown document; distinguish observed latency from assumptions.

## Phase 2 — Rolling encrypted vault saves

- Centralize edit notifications so typing, deletion, paste, undo/redo, Markdown assistance, and AI insertion all schedule persistence.
- Proposed timing: save after about 750 ms without an edit, with a maximum interval of about five seconds during continuous typing. Keep these values tunable during validation.
- Perform encrypted database writes asynchronously. Serialize writes and associate every immutable save snapshot with its vault/session, note identity, and edit version.
- Coalesce queued work to the latest applicable snapshot. An older completion must not overwrite newer content, clear a newer dirty state, or update the status of a different note.
- Ctrl+S flushes the latest edits immediately through the same save coordinator. Report success only after the database write succeeds; retain undo history.
- Show a quiet Saving / Saved state. Keep unsaved edits in memory on failure, display the failure, and permit retry without silently losing the draft.
- Include newly created untitled vault notes from their first edits. Keep disk documents on their existing explicit-save policy.
- Rolling saves update the current document; historical revisions are outside this phase. Sudden process termination may lose edits since the last completed save; do not imply per-keystroke durability.

Acceptance:

- Verify both pause-triggered and continuous-typing saves with controlled timing.
- Exercise rapid edits during slow writes, note changes, failures, and retries. Saved content and status must correspond to the correct note/version.
- Confirm persisted vault content follows the existing encryption path and no plaintext recovery files are introduced.
- Ctrl+S waits for the latest requested version to be persisted and leaves undo usable.

## Phase 3 — Safe Esc navigation and document transitions

Esc priority:

1. Close an active search or popup and return to the editor.
2. Otherwise, clear an active text selection while retaining editor focus.
3. Otherwise, leave the editor for the tree, showing the tree if hidden, and immediately flush pending vault edits.

- Enter in the tree opens the selected note in edit mode; Ctrl+E returns to the current document. Typing immediately inserts text once editor focus returns.
- Coordinate save completion before replacing a buffer, creating a new note, quitting, or locking the vault. Avoid blocking the UI thread while a write is pending.
- On save failure, preserve the draft and prevent a destructive transition until retry or an explicit user decision. Keep manual-save behavior for disk files, with save/discard/cancel when a transition would discard changes.
- Keep auto-lock integration consistent with existing save-failure behavior and prevent queued work from crossing vault sessions.

Acceptance:

- Complete a create/write/leave/reopen workflow without the mouse.
- Esc with a hidden tree reveals it; Esc with a popup or selection does not unexpectedly leave the editor.
- Slow or failed saves during Alt+N, document switching, quitting, and auto-lock cannot silently discard the current draft.
- Integrate the selection-specific Esc behavior once Phase 5 supplies persistent selection.

## Phase 4 — Markdown-aware typing

- Continue unordered bullets, ordered lists, checkboxes, and blockquotes on Enter. Preserve indentation and marker style; increment ordered-list numbers and reset continued checkboxes to unchecked.
- Enter on an empty item exits the list or steps out one nesting level.
- Tab and Shift+Tab nest and unnest list items. Outside lists, use ordinary indentation behavior.
- At an item's beginning, Backspace reduces nesting or removes the list prefix without unexpectedly deleting prose.
- Preserve indentation in fenced code blocks and suppress list assistance inside them.
- Apply assistance to interactive typing, not pasted blocks. Preserve pasted structure.
- Group automatic marker insertion with its triggering action for intuitive undo.
- Avoid whole-document renumbering or other unsolicited rewrites; initially only generate the next marker.

Acceptance:

- Cover bullets, multi-digit ordered lists, nested/mixed lists, empty items, checkboxes, blockquotes, fenced code, and Enter in the middle of an item.
- Verify Tab/Shift+Tab and Backspace behavior at nested and top-level items.
- Paste a Markdown block unchanged and undo an assisted Enter as one logical action.

## Phase 5 — Persistent selection and revision

- Add keyboard selection by character and word, plus persistent mouse selection.
- Typing, paste, Delete, and Backspace replace or remove the selected range; Esc clears it.
- Provide explicit copy/cut behavior using the keyboard map settled in Phase 1. Resolve the existing immediate drag-to-copy behavior deliberately.
- Permit internal selection and replacement in Eyes Only notes while retaining their clipboard restrictions.
- Support selection indentation/outdent using the Phase 4 rules where applicable.
- Keep cursor, selection, mouse coordinates, and wrapping consistent. Handle scrolling during selection and Unicode display widths.
- Decide whether a contained textarea extension/fork or a dedicated Bubble Tea editor component provides the maintainable path. Avoid expanding reliance on reflection into widget internals.

Acceptance:

- Select and rewrite a sentence without the mouse; repeat across wrapped rows and paragraph boundaries.
- Test reverse selection, scrolling, tabs, wide characters, combining characters, and resize.
- Replacement, cut, and indentation undo correctly and participate in rolling saves.
- Eyes Only editing works without enabling clipboard export.

## Phase 6 — Writing comfort and in-document search

- Offer configurable prose width, initially exploring 80–100 columns, with padding inside the existing editor pane. Retain a full-width option and visual-only wrapping.
- Keep several visible rows below the cursor during drafting; optionally support typewriter scrolling.
- Replace the disruptive find surface with an inline editor search bar, visible matches, and next/previous navigation. Add deliberate replacement actions using the selection model.
- Preserve document position through searching, resizing, and editor/preview transitions where practical.
- Make editor help reflect the active writing controls and save state; avoid a wider app redesign.

Acceptance:

- Evaluate long-form writing at narrow and wide terminal sizes, with the tree shown and hidden.
- Ensure padding and scrolling changes do not break mouse placement or selection.
- Search forward/backward, wrap through matches, cancel, and replace without losing position or unexpectedly changing text.

## Execution and validation

All phases are pending. Implement in order once implementation is requested, with Phase 3 selection integration completed during Phase 5. Keep changes reviewable by phase.

Use focused behavioral tests for input, save scheduling/races, transitions, and Markdown transformations. Run the existing Go test suite and build checks at phase completion; use the race detector for asynchronous save work. Perform an interactive terminal pass for actual shortcuts, clipboard paths, focus, cursor visibility, and responsiveness. Tests alone cannot establish writing comfort.

### Required close-out after every implementation phase

The user explicitly requires and authorizes these steps when implementation begins. A phase is not complete until all of them are done:

1. Update `README.md` to describe the behavior and shortcuts actually delivered by that phase. Preserve its existing irreverent voice, humor, and personality; do not replace it with generic product copy or a dry technical manual. Keep instructions accurate and remove superseded descriptions.
2. Run the phase's relevant checks and the existing Go test suite, then build both `weazlwrite` and `weazlwrite-setup`.
3. Update both installed binaries in `$HOME/.weazlwrite/bin/`, verifying that the command resolved on `PATH` points to the updated installation. Repo-local builds alone do not count.
4. Smoke-test the installed app using disposable documents and a test vault. Check launch/unlock, the phase's new editor behavior, typing and navigation, save/reopen persistence, preview, and clean exit. Exercise relevant clipboard and terminal shortcuts interactively. Include a non-destructive setup-binary sanity check without rewriting the user's configuration.
5. Record the checks performed and their results in this plan, along with any remaining limitations. If an environment limitation prevents an interactive check, report it explicitly and leave that check pending rather than claiming a successful smoke test.

Build and install commands:

```sh
go test ./...
go build -o weazlwrite ./cmd/weazlwrite
go build -o weazlwrite-setup ./cmd/weazlwrite-setup
install -m 755 weazlwrite "$HOME/.weazlwrite/bin/weazlwrite"
install -m 755 weazlwrite-setup "$HOME/.weazlwrite/bin/weazlwrite-setup"
```

This planning task creates only this document. It does not change code, build or install binaries, stage files, or create a commit.
