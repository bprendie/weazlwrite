package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/bprendie/weazlwrite/internal/tui"
)

func main() {
	cfg, cfgPath, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	var openPath string
	if len(os.Args) > 1 {
		openPath = os.Args[1]
	}

	p := tea.NewProgram(tui.New(cfg, cfgPath, openPath), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
