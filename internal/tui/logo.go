package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TYNEX logo in block ASCII art (thick block letters ██).
const logoArt = `
████████╗██╗   ██╗███╗   ██╗███████╗██╗  ██╗
╚══██╔══╝╚██╗ ██╔╝████╗  ██║██╔════╝╚██╗██╔╝
   ██║    ╚████╔╝ ██╔██╗ ██║█████╗   ╚███╔╝ 
   ██║     ╚██╔╝  ██║╚██╗██║██╔══╝   ██╔██╗ 
   ██║      ██║   ██║ ╚████║███████╗██╔╝ ██╗
   ╚═╝      ╚═╝   ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝
`

// Gradient colors for logo (bottom to top):
// Blue(#3B82F6) → Green(#10B981) → Yellow(#F59E0B) → Red-pink(#EF4444) → Purple(#8B5CF6)
var gradientColors = []lipgloss.Color{
	lipgloss.Color("#3B82F6"), // Blue (bottom)
	lipgloss.Color("#10B981"), // Green
	lipgloss.Color("#F59E0B"), // Yellow (middle)
	lipgloss.Color("#EF4444"), // Red-pink
	lipgloss.Color("#8B5CF6"), // Purple (top)
}

// BannerOptions configures the banner display.
type BannerOptions struct {
	ShowLogo    bool
	ShowDivider bool
	Provider    string
	Model       string
	Version     string
}

// theme colors
var (
	colorGray       = lipgloss.Color("#666666")
	colorPurple     = lipgloss.Color("#7C3AED")
	colorGreen      = lipgloss.Color("#10B981")
	colorYellow     = lipgloss.Color("#F59E0B")
	colorBlue       = lipgloss.Color("#3B82F6")
)

// noColor returns true if colors should be disabled (NO_COLOR env var).
func noColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// lerpColor performs linear interpolation between two RGB hex colors.
// ratio=0 returns color1, ratio=1 returns color2.
func lerpColor(c1, c2 lipgloss.Color, ratio float64) lipgloss.Color {
	r1, g1, b1 := parseHex(string(c1))
	r2, g2, b2 := parseHex(string(c2))

	r := int(float64(r1) + ratio*float64(r2-r1))
	g := int(float64(g1) + ratio*float64(g2-g1))
	b := int(float64(b1) + ratio*float64(b2-b1))

	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", clamp(r), clamp(g), clamp(b)))
}

// parseHex parses a hex color string (#RRGGBB) into RGB components.
func parseHex(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r := 0
	g := 0
	b := 0
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// clamp restricts a value to the range [0, 255].
func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// RenderGradientLogo returns the TYNEX logo with gradient color scheme.
// Each line gets an interpolated color from the gradient palette (bottom to top).
// ratio := currentRow/totalRows for RGB lerp interpolation.
func RenderGradientLogo() string {
	if noColor() {
		return logoArt
	}

	lines := strings.Split(strings.TrimRight(logoArt, "\n"), "\n")
	var coloredLines []string

	totalRows := len(lines)
	numColors := len(gradientColors)

	// Assign gradient colors across the lines using lerp interpolation (bottom to top)
	for i, line := range lines {
		// ratio: 0 at bottom, 1 at top
		ratio := float64(totalRows-1-i) / float64(max(totalRows-1, 1))

		// Find which two gradient colors to interpolate between
		colorPos := ratio * float64(numColors-1)
		idx := int(colorPos)
		nextIdx := idx + 1
		if nextIdx >= numColors {
			nextIdx = numColors - 1
			idx = numColors - 1
		}
		frac := colorPos - float64(idx)

		color := gradientColors[idx]
		if frac > 0 && nextIdx < numColors {
			color = lerpColor(gradientColors[idx], gradientColors[nextIdx], frac)
		}

		style := lipgloss.NewStyle().Foreground(color)
		coloredLines = append(coloredLines, style.Render(line))
	}

	return strings.Join(coloredLines, "\n")
}

// RenderBanner renders the full Tynex banner with logo, header info, current directory, and divider.
func RenderBanner(opts BannerOptions) string {
	var sb strings.Builder

	if opts.ShowLogo {
		sb.WriteString(RenderGradientLogo())
		sb.WriteString("\n")
	}

	// ── Header line: version | active provider/model | current directory ──
	var headerParts []string
	if opts.Version != "" {
		headerParts = append(headerParts, fmt.Sprintf("v%s", opts.Version))
	}
	if opts.Provider != "" || opts.Model != "" {
		info := ""
		if opts.Provider != "" {
			info = opts.Provider
		}
		if opts.Model != "" {
			if info != "" {
				info += "/"
			}
			info += opts.Model
		}
		headerParts = append(headerParts, info)
	}

	// Current working directory
	if cwd, err := os.Getwd(); err == nil {
		home, _ := os.UserHomeDir()
		displayPath := cwd
		if home != "" && strings.HasPrefix(cwd, home) {
			displayPath = "~" + strings.TrimPrefix(cwd, home)
		}
		headerParts = append(headerParts, displayPath)
	}

	headerText := strings.Join(headerParts, "  ·  ")

	if !noColor() {
		grayStyle := lipgloss.NewStyle().Foreground(colorGray)
		sb.WriteString(grayStyle.Render(headerText))
	} else {
		sb.WriteString(headerText)
	}
	sb.WriteString("\n")

	// ── Full-width divider ──
	if opts.ShowDivider {
		width := 80
		divider := strings.Repeat("─", width)
		if !noColor() {
			dividerStyle := lipgloss.NewStyle().Foreground(colorGray)
			sb.WriteString(dividerStyle.Render(divider))
		} else {
			sb.WriteString(divider)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderPromptPrefix returns the user input prompt prefix.
func RenderPromptPrefix() string {
	if noColor() {
		return ">>> "
	}
	style := lipgloss.NewStyle().Foreground(colorPurple).Bold(true)
	return style.Render("> ")
}

// RenderToolCall formats a tool call display line.
// Format: ● tool_name(arguments)
func RenderToolCall(toolName, args string) string {
	if noColor() {
		return fmt.Sprintf("  ● %s(%s)", toolName, args)
	}

	dot := lipgloss.NewStyle().Foreground(colorGreen).Render("●")
	name := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(toolName)
	argStyle := lipgloss.NewStyle().Foreground(colorGray).Italic(true).Render(args)

	return fmt.Sprintf("  %s %s(%s)", dot, name, argStyle)
}

// RenderStatusBar renders the bottom status bar.
func RenderStatusBar(model string) string {
	if noColor() {
		return fmt.Sprintf("── Model: %s ──", model)
	}

	modelStyle := lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(colorGray)

	return statusStyle.Render("─ ") + modelStyle.Render(model) + statusStyle.Render(" ─")
}

// RenderAssistantPrefix returns the Tynex response prefix.
func RenderAssistantPrefix() string {
	if noColor() {
		return "Tynex: "
	}
	style := lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	return style.Render("Tynex: ")
}
