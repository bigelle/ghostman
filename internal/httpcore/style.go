package httpcore

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/lipgloss"
)

type Method string

func (m Method) String() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("FFFFFF")).
		Bold(true).
		Padding(0, 1)
	switch m {
	case "GET":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#16a34a",
				ANSI256:   "28",
				ANSI:      "2",
			},
		)
		return style.Render(string(m))
	case "POST":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#3b82f6",
				ANSI256:   "33",
				ANSI:      "4",
			},
		)
		return style.Render(string(m))
	case "PUT":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f59e0b",
				ANSI256:   "214",
				ANSI:      "3",
			},
		)
		return style.Render(string(m))
	case "PATCH":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#8b5cf6",
				ANSI256:   "99",
				ANSI:      "5",
			},
		)
		return style.Render(string(m))
	case "DELETE":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#ef4444",
				ANSI256:   "196",
				ANSI:      "1",
			},
		)
		return style.Render(string(m))
	case "HEAD":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#06b6d4",
				ANSI256:   "31",
				ANSI:      "6",
			},
		)
		return style.Render(string(m))
	case "OPTIONS":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#84cc16",
				ANSI256:   "112",
				ANSI:      "10",
			},
		)
		return style.Render(string(m))
	case "TRACE":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#64748b",
				ANSI256:   "244",
				ANSI:      "8",
			},
		)
		return style.Render(string(m))
	case "CONNECT":
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f97316",
				ANSI256:   "202",
				ANSI:      "9",
			},
		)
		return style.Render(string(m))
	default:
		return style.Render(string(m))
	}
}

type Status int

func (s Status) String() string {
	if s < 100 || 600 <= s {
		return fmt.Sprintf("%d %s", s, http.StatusText(int(s)))
	}

	style := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("#FFFFFF"))

	if 100 <= s && s < 200 {
		style = style.Foreground(lipgloss.Color("#000000")).
			Background(
				lipgloss.CompleteColor{
					TrueColor: "#ECEFF1",
					ANSI256:   "254",
					ANSI:      "7",
				},
			)
	}
	if 200 <= s && s < 300 {
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#16a34a",
				ANSI256:   "28",
				ANSI:      "2",
			},
		)
	}
	if 300 <= s && s < 400 {
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#06b6d4",
				ANSI256:   "31",
				ANSI:      "6",
			},
		)
	}
	if 400 <= s && s < 500 {
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#f97316",
				ANSI256:   "202",
				ANSI:      "9",
			},
		)
	}
	if 500 <= s && s < 600 {
		style = style.Background(
			lipgloss.CompleteColor{
				TrueColor: "#ef4444",
				ANSI256:   "196",
				ANSI:      "1",
			},
		)
	}

	return style.Render(fmt.Sprintf("%d %s", s, http.StatusText(int(s))))
}
