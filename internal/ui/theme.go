package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha palette.
var (
	colText     = lipgloss.Color("#cdd6f4")
	colSubtext  = lipgloss.Color("#a6adc8")
	colOverlay  = lipgloss.Color("#6c7086")
	colSurface  = lipgloss.Color("#313244")
	colMauve    = lipgloss.Color("#cba6f7")
	colBlue     = lipgloss.Color("#89b4fa")
	colGreen    = lipgloss.Color("#a6e3a1")
	colYellow   = lipgloss.Color("#f9e2af")
	colRed      = lipgloss.Color("#f38ba8")
	colLavender = lipgloss.Color("#b4befe")
)

// icons holds the glyphs used in the UI.
type icons struct {
	Open, Done, Star, Collapsed, Expanded, Sep, Ellipsis string
}

var unicodeIcons = icons{"○", "✓", "★", "▸", "▾", "›", "…"}
var asciiIcons = icons{"[ ]", "[x]", "*", ">", "v", ">", "~"}

// theme bundles every Lip Gloss style used by the UI.
type theme struct {
	icons icons

	paneActive, paneIdle lipgloss.Style
	title, titleIdle     lipgloss.Style
	row, rowSel          lipgloss.Style
	rowDim, rowDimSel    lipgloss.Style
	done, doneSel        lipgloss.Style
	star, starSel        lipgloss.Style
	muted, mutedSel      lipgloss.Style
	hint, hintKey        lipgloss.Style
	err                  lipgloss.Style
	overlay              lipgloss.Style
}

// newTheme builds the default theme. With ascii set, icons fall back to ASCII.
func newTheme(ascii bool) theme {
	t := theme{icons: unicodeIcons}
	if ascii {
		t.icons = asciiIcons
	}
	bd := lipgloss.RoundedBorder()
	if ascii {
		bd = lipgloss.NormalBorder()
	}
	border := func(c lipgloss.Color) lipgloss.Style {
		return lipgloss.NewStyle().Border(bd).BorderForeground(c)
	}
	t.paneActive = border(colMauve)
	t.paneIdle = border(colSurface)
	t.title = lipgloss.NewStyle().Bold(true).Foreground(colMauve)
	t.titleIdle = lipgloss.NewStyle().Bold(true).Foreground(colOverlay)

	sel := func(s lipgloss.Style) lipgloss.Style { return s.Background(colSurface) }
	t.row = lipgloss.NewStyle().Foreground(colText)
	t.rowSel = sel(t.row).Bold(true).Foreground(colLavender)
	t.rowDim = lipgloss.NewStyle().Foreground(colSubtext)
	t.rowDimSel = sel(t.rowDim).Bold(true)
	t.done = lipgloss.NewStyle().Foreground(colOverlay).Strikethrough(true)
	t.doneSel = sel(t.done)
	t.star = lipgloss.NewStyle().Foreground(colYellow)
	t.starSel = sel(t.star)
	t.muted = lipgloss.NewStyle().Foreground(colOverlay)
	t.mutedSel = sel(t.muted)
	t.hint = lipgloss.NewStyle().Foreground(colOverlay)
	t.hintKey = lipgloss.NewStyle().Foreground(colBlue)
	t.err = lipgloss.NewStyle().Foreground(colRed)
	t.overlay = lipgloss.NewStyle().Border(bd).BorderForeground(colBlue).Padding(1, 2)
	return t
}

// asciiFromEnv reports whether TASK_TUI_ASCII asks for the ASCII fallback.
func asciiFromEnv() bool {
	v := os.Getenv("TASK_TUI_ASCII")
	return v != "" && v != "0" && v != "false"
}
