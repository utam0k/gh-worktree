package ui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/utam0k/gh-worktree/internal/github"
)

const (
	keyUp        = 65
	keyDown      = 66
	keyEnter     = 13
	keyCtrlC     = 3
	keyBackspace = 127
	keyEscape    = 27
)

type PRSelector struct {
	prs          []github.PRListItem
	filtered     []github.PRListItem
	selected     int
	searchQuery  string
	termHeight   int
	displayStart int
}

func NewPRSelector(prs []github.PRListItem) *PRSelector {
	filtered := make([]github.PRListItem, len(prs))
	copy(filtered, prs)

	return &PRSelector{
		prs:        prs,
		filtered:   filtered,
		selected:   0,
		termHeight: getTerminalHeight() - 6, // Leave space for header and search
	}
}

func (s *PRSelector) SelectPR() (string, error) {
	// Set terminal to raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)

	// Hide cursor
	draw("\033[?25l")
	defer draw("\033[?25h") // Show cursor on exit

	for {
		s.render()

		// Read key
		key := make([]byte, 3)
		n, _ := os.Stdin.Read(key)

		if n == 1 {
			switch key[0] {
			case keyEnter:
				if len(s.filtered) > 0 {
					return fmt.Sprintf("%d", s.filtered[s.selected].Number), nil
				}
			case keyCtrlC:
				return "", fmt.Errorf("cancelled")
			case keyBackspace:
				if len(s.searchQuery) > 0 {
					s.searchQuery = s.searchQuery[:len(s.searchQuery)-1]
					s.updateFilter()
				}
			default:
				// Regular character
				if key[0] >= 32 && key[0] < 127 {
					s.searchQuery += string(key[0])
					s.updateFilter()
				}
			}
		} else if n == 3 && key[0] == keyEscape && key[1] == '[' {
			switch key[2] {
			case keyUp:
				if s.selected > 0 {
					s.selected--
					s.adjustScroll()
				}
			case keyDown:
				if s.selected < len(s.filtered)-1 {
					s.selected++
					s.adjustScroll()
				}
			}
		}
	}
}

func (s *PRSelector) updateFilter() {
	s.filtered = []github.PRListItem{}
	query := strings.ToLower(s.searchQuery)

	for _, pr := range s.prs {
		if query == "" ||
			strings.Contains(strings.ToLower(pr.Title), query) ||
			strings.Contains(strings.ToLower(pr.HeadRefName), query) ||
			strings.Contains(strings.ToLower(pr.Author.Login), query) ||
			strings.Contains(fmt.Sprintf("%d", pr.Number), query) {
			s.filtered = append(s.filtered, pr)
		}
	}

	// Reset selection if out of bounds
	s.selected = 0
	s.displayStart = 0
}

func (s *PRSelector) adjustScroll() {
	if s.selected < s.displayStart {
		s.displayStart = s.selected
	} else if s.selected >= s.displayStart+s.termHeight {
		s.displayStart = s.selected - s.termHeight + 1
	}
}

func (s *PRSelector) render() {
	var b strings.Builder

	// Clear screen and move to top
	b.WriteString("\033[2J\033[H")

	// Header
	b.WriteString("\033[1mSelect a Pull Request\033[0m\n")
	b.WriteString("\033[90m(Use ↑↓ to navigate, type to search, Enter to select, Ctrl-C to cancel)\033[0m\n")

	// Search box
	fmt.Fprintf(&b, "\n\033[1mSearch:\033[0m %s_\n\n", s.searchQuery)

	// PR list
	if len(s.filtered) == 0 {
		b.WriteString("\033[90mNo matching pull requests\033[0m\n")
		draw(b.String())
		return
	}

	// Display visible items
	end := s.displayStart + s.termHeight
	if end > len(s.filtered) {
		end = len(s.filtered)
	}

	for i := s.displayStart; i < end; i++ {
		pr := s.filtered[i]

		// Highlight selected item
		if i == s.selected {
			b.WriteString("\033[7m") // Reverse video
		}

		// Format PR info
		title := pr.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		branch := pr.HeadRefName
		if len(branch) > 20 {
			branch = branch[:17] + "..."
		}

		fmt.Fprintf(&b, " #%-5d %-50s \033[32m%s\033[0m \033[90m@%s\033[0m",
			pr.Number, title, branch, pr.Author.Login)

		if i == s.selected {
			b.WriteString("\033[0m") // Reset
		}
		b.WriteString("\n")
	}

	// Show scroll indicator if needed
	if end < len(s.filtered) {
		fmt.Fprintf(&b, "\n\033[90m... and %d more\033[0m", len(s.filtered)-end)
	}

	draw(b.String())
}

// draw writes to stderr rather than stdout so that only the selected value
// reaches stdout, e.g. when run as `cd $(gh worktree cd)`. Raw mode turns off
// output processing, so each newline needs an explicit carriage return.
func draw(s string) {
	fmt.Fprint(os.Stderr, strings.ReplaceAll(s, "\n", "\r\n"))
}

func getTerminalHeight() int {
	_, height, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || height < 10 {
		return 20 // Default height
	}
	return height
}
