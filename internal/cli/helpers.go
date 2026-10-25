package cli

import (
	"flag"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseFlagSet(fs *flag.FlagSet, args []string) error {
	return fs.Parse(normalizeFlagArgs(fs, args))
}

func normalizeFlagArgs(fs *flag.FlagSet, args []string) []string {
	type boolFlag interface {
		IsBoolFlag() bool
	}

	var flags []string
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "-") && arg != "-" {
			trimmed := strings.TrimLeft(arg, "-")
			name, hasValue := trimmed, false
			if before, _, ok := strings.Cut(trimmed, "="); ok {
				name = before
				hasValue = true
			}

			if f := fs.Lookup(name); f != nil {
				flags = append(flags, arg)
				if !hasValue {
					if bf, ok := f.Value.(boolFlag); ok && bf.IsBoolFlag() {
						continue
					}
					if i+1 < len(args) {
						flags = append(flags, args[i+1])
						i++
						continue
					}
				}
				continue
			}

			// Unknown flag: keep it as flag so Parse() can error.
			flags = append(flags, arg)
			continue
		}

		positionals = append(positionals, arg)
	}

	return append(flags, positionals...)
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
