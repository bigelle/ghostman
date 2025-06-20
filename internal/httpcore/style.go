package httpcore

import "github.com/charmbracelet/lipgloss"


type Method string

func (m Method) String() string {
	baseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("FFFFFF")).
		Bold(true).
		Padding(0, 1)
	switch m {
	case "GET":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#16a34a",
				ANSI256:   "28",
				ANSI:      "2",
			},
		)
		return style.Render(string(m))
	case "POST":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#3b82f6",
				ANSI256:   "33",
				ANSI:      "4",
			},
		)
		return style.Render(string(m))
	case "PUT":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f59e0b",
				ANSI256:   "214",
				ANSI:      "3",
			},
		)
		return style.Render(string(m))
	case "PATCH":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#8b5cf6",
				ANSI256:   "99",
				ANSI:      "5",
			},
		)
		return style.Render(string(m))
	case "DELETE":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#ef4444",
				ANSI256:   "196",
				ANSI:      "1",
			},
		)
		return style.Render(string(m))
	case "HEAD":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#06b6d4",
				ANSI256:   "31",
				ANSI:      "6",
			},
		)
		return style.Render(string(m))
	case "OPTIONS":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#84cc16",
				ANSI256:   "112",
				ANSI:      "10",
			},
		)
		return style.Render(string(m))
	case "TRACE":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#64748b",
				ANSI256:   "244",
				ANSI:      "8",
			},
		)
		return style.Render(string(m))
	case "CONNECT":
		style := baseStyle.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f97316",
				ANSI256:   "202",
				ANSI:      "9",
			},
		)
		return style.Render(string(m))
	default:
		return baseStyle.Render(string(m))
	}
}

