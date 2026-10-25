package ui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/utam0k/gh-worktree/internal/worktree"
)

type WorktreeSelector struct {
	worktrees    []worktree.WorktreeInfo
	filtered     []worktree.WorktreeInfo
	selected     int
	searchQuery  string
	termHeight   int
	displayStart int
}

func NewWorktreeSelector(worktrees []worktree.WorktreeInfo) *WorktreeSelector {
	filtered := make([]worktree.WorktreeInfo, len(worktrees))
	copy(filtered, worktrees)

	return &WorktreeSelector{
		worktrees:  worktrees,
		filtered:   filtered,
		selected:   0,
		termHeight: getTerminalHeight() - 6, // Leave space for header and search
	}
}

func (s *WorktreeSelector) SelectWorktree() (*worktree.WorktreeInfo, error) {
	// Set terminal to raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
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
					return &s.filtered[s.selected], nil
				}
			case keyCtrlC:
				return nil, fmt.Errorf("cancelled")
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

func (s *WorktreeSelector) updateFilter() {
	s.filtered = []worktree.WorktreeInfo{}
	query := strings.ToLower(s.searchQuery)

	for _, wt := range s.worktrees {
		if query == "" ||
			strings.Contains(strings.ToLower(wt.PRNumber), query) ||
			strings.Contains(strings.ToLower(wt.Branch), query) ||
			strings.Contains(strings.ToLower(wt.Remote), query) ||
			strings.Contains(strings.ToLower(wt.Path), query) {
			s.filtered = append(s.filtered, wt)
		}
	}

	// Reset selection if out of bounds
	s.selected = 0
	s.displayStart = 0
}

func (s *WorktreeSelector) adjustScroll() {
	if s.selected < s.displayStart {
		s.displayStart = s.selected
	} else if s.selected >= s.displayStart+s.termHeight {
		s.displayStart = s.selected - s.termHeight + 1
	}
}

func (s *WorktreeSelector) render() {
	var b strings.Builder

	// Clear screen and move to top
	b.WriteString("\033[2J\033[H")

	// Header
	b.WriteString("\033[1mSelect a Worktree\033[0m\n")
	b.WriteString("\033[90m(Use ↑↓ to navigate, type to search, Enter to select, Ctrl-C to cancel)\033[0m\n")

	// Search box
	fmt.Fprintf(&b, "\n\033[1mSearch:\033[0m %s_\n\n", s.searchQuery)

	// Worktree list
	if len(s.filtered) == 0 {
		b.WriteString("\033[90mNo matching worktrees\033[0m\n")
		draw(b.String())
		return
	}

	// Display visible items
	end := s.displayStart + s.termHeight
	if end > len(s.filtered) {
		end = len(s.filtered)
	}

	for i := s.displayStart; i < end; i++ {
		wt := s.filtered[i]

		// Highlight selected item
		if i == s.selected {
			b.WriteString("\033[7m") // Reverse video
		}

		// Format worktree info based on type
		if wt.Type == "pr" {
			remote := wt.Remote
			if remote == "" {
				remote = "-"
			}

			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			if len(branch) > 20 {
				branch = branch[:17] + "..."
			}

			fmt.Fprintf(&b, " PR #%-5s %-10s %-20s %s",
				wt.PRNumber, remote, branch, wt.DisplayPath)
		} else {
			// Work worktree
			branch := wt.Branch
			if len(branch) > 30 {
				branch = branch[:27] + "..."
			}

			fmt.Fprintf(&b, " [WORK] %-30s %s", branch, wt.DisplayPath)
		}

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
