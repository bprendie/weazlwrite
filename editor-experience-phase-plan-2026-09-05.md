# Editor Experience Phase Plan

Date: September 5, 2026
Status: Phases 1–6 implemented and validated, including quiet rolling saves. User accepted the result and authorized commit and push on September 5, 2026; rollback checkpoint is `423638a`.

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

Keep code files below the soft ceiling of 300 lines. Split by responsibility rather than growing large editor or model files; audit source-file lengths at each close-out with `./scripts/check-loc.sh`.

Phases 1–6 are complete, including selection-first Esc. User review is complete and commit/push is authorized; the rollback checkpoint is unchanged.

Use focused behavioral tests for input, save scheduling/races, transitions, and Markdown transformations. Run the existing Go test suite and build checks at phase completion; use the race detector for asynchronous save work. Perform an interactive terminal pass for actual shortcuts, clipboard paths, focus, cursor visibility, and responsiveness. Tests alone cannot establish writing comfort.

### Required close-out after every implementation phase

The user explicitly requires and authorizes these steps when implementation begins. A phase is not complete until all of them are done:

1. Update `README.md` to describe the behavior and shortcuts actually delivered by that phase. Preserve its existing irreverent voice, humor, and personality; do not replace it with generic product copy or a dry technical manual. Keep instructions accurate and remove superseded descriptions. Do not mention phases in the README; keep implementation tracking in this plan.
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

The original planning request created only this document. The user subsequently authorized the first safe implementation phases, a checkpoint commit, and the per-phase README/build/install/smoke-test workflow recorded below.

## Execution log — September 5, 2026

### Phase 1 completed

- Rollback checkpoint: `423638a` (`Checkpoint before editor experience phases`). Existing untracked personal notes and screenshot were excluded.
- Delivered editor-owned Tab/Shift+Tab (four-space indentation), clipboard and bracketed paste with tab preservation, stale clipboard rejection across documents, intent-based undo boundaries, cursor-message routing, and deferred hidden preview rendering.
- Shortcut audit: Ctrl+Y and Alt+Z redo; Alt+M toggles mouse capture; Alt+D deletes the next word; Alt+S saves to a filesystem path. Ctrl+C remains quit until selection work supplies copy semantics; Ctrl+Q will provide an unambiguous quit alternative during transition work. Existing word-movement and Home/End bindings remain available.
- README and command help updated to delivered behavior, preserving the README voice and omitting phase terminology.
- Validation: `go test ./...` passed; both binaries built and installed; installed files match repo builds and PATH resolves to `$HOME/.weazlwrite/bin`.
- Installed-app smoke test in an isolated tmux terminal: created/unlocked a disposable vault, edited a disposable disk document, indented/outdented, pasted via Ctrl+V, undid/redid, saved, previewed, returned via keyboard, quit, reopened, and bracket-pasted multiline tabbed Markdown. Persisted text matched expectations. The clipboard utility was stubbed with disposable text to avoid reading or replacing the user's clipboard; actual desktop clipboard integration remains unverified.
- Installed setup smoke test passed using disposable configuration and an unavailable localhost provider to exercise manual configuration fallback.
- A benchmark with 1,000 Markdown paragraphs (~49 KB) measured ~106 ms per insert/render/delete cycle on this machine. Hidden preview work is removed, but full-buffer layout remains a long-document performance limitation for later optimization; this is not a claim of instantaneous typing at all document sizes.

### Phase 2 completed

- Implemented serialized asynchronous encrypted saves after 750 ms of quiet or five seconds of continuous typing, immutable note/version snapshots, coalescing after slow writes, immediate Ctrl+S flush, saved-state reporting, and explicit retry after failure.
- Central edit tracking covers typing, indentation, paste, undo/redo, AI insertion, and new untitled vault notes. Disk files retain explicit saving. Existing vault payload encryption and metadata format are unchanged; no revision-history or plaintext recovery files were added.
- Included the prerequisite vault transition barrier now: note/path mutations, quit, and auto-lock drain writes before proceeding. This prevents installing background saves without safe lifecycle handling. Phase 3 adds disk draft prompts and completes Esc behavior.
- Validation: full Go suite passed; race detector passed for TUI and storage. Controlled tests cover debounce/max interval, slow writes with continued editing, retry, stale messages, encryption of document payloads, undo preservation, new-note/quit transitions, and auto-lock.
- README describes rolling saves without phase terminology. Both binaries rebuilt and installed, with byte-for-byte installation verification.
- Installed tmux smoke: typed into a disposable vault without Ctrl+S and independently decrypted its persisted payload to confirm the save; verified Saved status, continued typing through the five-second interval, immediate Alt+N and Ctrl+Q flushes, reopening a vault note, preview/edit navigation, and clean exit. Setup smoke passed with disposable config.

### Phase 3 completed

- Esc closes search/help first; from the editor it reveals and focuses the tree. Ctrl+E returns to the current draft; Enter opens the selected tree note. Persistent editable selection remains Phase 5, including its Esc-clear behavior.
- Vault departure barriers save the latest version before new/open/path mutations/quit/lock, retain the draft on failure, and allow Esc to cancel a waiting departure while saving continues. Ctrl+Q is an explicit quit alternative; Ctrl+C retains its legacy quit behavior pending selection work.
- Disk drafts offer save/discard/cancel before replacement or quit. Esc to the tree alone neither saves nor discards them. Save failures retain the draft and prompt. Detached buffers left by note deletion require explicit intent before saving under a new name; auto-lock is delayed for such a buffer instead of silently recreating the deleted note.
- Auto-lock tests cover both an in-flight save and a clean expired vault. An expired clean vault locks before accepting new input; a failed lock save refreshes activity so the user can resume editing and retry instead of becoming trapped in repeated save attempts.
- Validation: `go test ./...` passed; `go test -race ./internal/tui ./internal/storage` passed. Focused tests cover hidden-tree Esc, search cancellation, save/discard/cancel, save failures, note switching, cancelled pending transitions, deleted-note buffers, and lock/save ordering.
- Both binaries rebuilt and installed after the final changes; byte-for-byte checks and PATH resolution confirm the installed copies. README and command help reflect the new behavior, with the existing README voice and no phase references.
- Installed-app smoke tests used disposable vaults and files in an isolated tmux terminal: hidden-tree Esc, return to writing, search cancellation, cancel Alt+N, save-on-quit, reopen, discard-on-quit, preview/edit, Tab/Shift+Tab, rolling vault save, immediate quit flush, independent encrypted-payload verification, and clean exits all passed. Installed setup smoke passed with disposable configuration. Escape was sent as a separate key event with a short pause so terminal escape-prefix parsing did not combine it with the next shortcut.
- Remaining limitations: actual desktop clipboard service integration has not been tested (the clipboard command was isolated with test data); long-document layout cost remains as measured in Phase 1. Bullet/numbered-list continuation and nesting have not been implemented yet; Phase 4 is next.

### Phase 4 completed

- Added bullet/ordered-list/checkbox/blockquote continuation, empty-item exit or unnest, list nesting with Tab, prefix-aware Backspace, and fenced-code indentation. Paste bypasses assistance and assisted edits remain single undo actions.
- Full Go suite passed, including new cases for multi-digit numbers, nested lists/quotes, checked tasks, fences, splitting an item mid-line, indentation, and unchanged pasted Markdown.
- README and command help updated to delivered behavior without phase mentions. Both binaries built and installed.
- Installed tmux smoke passed: bullet continuation, list indent/outdent, empty-item exit, numbering from 9 to 10, fenced-code indentation, paste, save/reopen, preview/edit, clean exit, and setup with disposable configuration.

### Phase 5 completed

- Added persistent keyboard/mouse selection, word and document selection, replacement by typing/paste/delete, selection indentation, explicit copy/cut, and selection-first Esc. Ctrl+C copies in the editor; Ctrl+Q quits. Clipboard failure leaves a cut intact; Eyes Only permits internal replacement but blocks copy/cut.
- A contained MIT-licensed fork of Bubbles textarea v0.21.0 now exposes buffer coordinates and renders highlights using the same wrapping as navigation and mouse hit testing. Removed viewport reflection and the separate drag renderer. Wrapping and character-selection movement preserve combining sequences and joined emoji.
- Split the copied component and pre-existing oversized source/test files by responsibility. All Go files are below 300 lines; the small importer/LLM/vault-flow splits are structural only.
- Full suite and TUI/editor-buffer race checks passed. Installed both rebuilt binaries and updated README/help without phase mentions.
- Installed tmux smoke passed: select/replace, Ctrl+A, Ctrl+C/X/V using isolated clipboard utilities, undo, joined-emoji selection/deletion, save/reopen, preview/edit, clean quit, and setup with disposable config. No user clipboard contents were read or changed.


### Phase 6 completed

- Added centered prose width (90 columns by default; Alt+W cycles full/80/90/100), configurable scroll margin, and optional typewriter scrolling (Alt+T). Width/typewriter preferences persist to configuration. Preview retains its own width and approximates the current source position.
- Inline Ctrl+F keeps the document visible, highlights matches, and supports next/previous navigation. Tab opens replacement; Enter replaces one match as a single undo action. F2 selects the match for direct rewriting; Esc restores the original caret/scroll position while retaining explicit replacements. Shift+F3 and Alt+F3 navigate backward.
- Shared wrapping preserves graphemes and whitespace; cached row positions and rendering only visible rows reduce the 1,000-paragraph (~49 KB) insert/render/delete benchmark from approximately 106 ms to 14 ms on this machine. This is a measured workload, not a guarantee for arbitrary document sizes.
- Corrected panel padding dimensions so visible columns agree with mouse hit testing. Final installed testing also caught multiline selection-paste caret placement with terminal carriage returns; replacement now uses the buffer's insertion position, with CRLF normalization and regression coverage for LF/CR/CRLF.
- Validation: `go test ./...`, `go test -race ./internal/tui ./internal/editorbuffer ./internal/storage`, `git diff --check`, and `./scripts/check-loc.sh` passed. All code files remain below 300 lines. Tests include narrow/wide layouts, sidebar bounds, Unicode selection, viewport scrolling, search navigation/replacement/cancellation, and cache invalidation.
- README/help updated to delivered behavior while preserving the README voice and keeping phase terminology out. Both binaries rebuilt and installed after the final fixes; byte comparisons and PATH resolution confirm the installed copies.
- Installed tmux smoke checks passed with disposable documents/configuration/vault: width/typewriter preferences, inline find/replace and backward navigation, mouse selection in the padded pane, multiline paste followed by typing, nested quoted fences, disk save/reopen, preview/edit, rolling encrypted vault persistence after typing and replacement, clean quit, and setup fallback. Vault payloads were independently decrypted to verify exact saved text.
- Remaining limitation: desktop clipboard service integration remains unverified; copy/cut/paste command paths were tested with isolated utilities and disposable text. Existing personal notes and screenshot were untouched. No implementation commit was created.

### Follow-up — quiet rolling saves

- User feedback: changing save indicators distract during typing. Background vault saves now leave the status text alone and omit the alternating Saving/Saved label and dirty asterisk for attached vault drafts. Existing debounce/max-interval timing is unchanged.
- Ctrl+S explicitly requests a completion acknowledgement, including when pressed during an older in-flight write; confirmation waits for the latest version. Save failures, pending-departure help, unsaved disk markers, and detached draft markers remain visible.
- README updated in its existing voice without phase mentions. Full Go suite, source ceiling check, and whitespace checks passed. Focused tests cover stable status across typing/write/completion, encrypted persistence, manual confirmation after coalesced writes, errors, and unsaved markers.
- Both binaries rebuilt and installed; byte comparisons and PATH checks passed. Installed tmux smoke confirmed silent autosave, independently decrypted the exact saved draft, verified Ctrl+S acknowledgement and clean quit, and exercised setup using disposable configuration.
