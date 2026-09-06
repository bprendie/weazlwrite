This package is a contained fork of Charmbracelet Bubbles textarea v0.21.0, under the included MIT license. Upstream: https://github.com/charmbracelet/bubbles/tree/v0.21.0/textarea

WeazlWrite additions expose cursor/viewport coordinates and render selection/search highlights against the same wrapping model as the buffer. App-level editing, Markdown assistance, undo, and persistence remain in internal/tui.

The fork also preserves grapheme boundaries and pasted newline formats, caches wrapped row positions, renders visible rows, and supports scroll margins and optional typewriter positioning. Source is split by responsibility to stay below the project’s 300-line ceiling.
