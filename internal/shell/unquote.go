package shell

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// This is borrowed from rig v2 until k0sctl is updated to use it

var (
	builderPool = sync.Pool{
		New: func() any {
			return &strings.Builder{}
		},
	}

	// ErrMismatchedQuotes is returned when the input string has mismatched quotes when unquoting.
	ErrMismatchedQuotes = errors.New("mismatched quotes")

	// ErrTrailingBackslash is returned when the input string ends with a trailing backslash.
	ErrTrailingBackslash = errors.New("trailing backslash")
)

// Unquote is a mostly POSIX compliant implementation of unquoting a string the same way a shell would.
// Variables and command substitutions are not handled.
// Additionally handles \xNN hex escape sequences commonly found in systemd service files.
func Unquote(input string) (string, error) { //nolint:cyclop
	sb, ok := builderPool.Get().(*strings.Builder)
	if !ok {
		sb = &strings.Builder{}
	}
	defer builderPool.Put(sb)
	defer sb.Reset()

	var inDoubleQuotes, inSingleQuotes, isEscaped bool

	for i := 0; i < len(input); i++ {
		currentChar := input[i]

		if isEscaped {
			sb.WriteByte(currentChar)
			isEscaped = false
			continue
		}

		switch currentChar {
		case '\\':
			if inSingleQuotes {
				sb.WriteByte(currentChar)
				continue
			}

			if i == len(input)-1 {
				return "", fmt.Errorf("unquote `%q`: %w", input, ErrTrailingBackslash)
			}

			nextChar := input[i+1]

			// Handle \xNN hex escape sequences (e.g. \x20 for space).
			// These are commonly found in systemd service files.
			if nextChar == 'x' && i+3 < len(input) {
				if b, err := hex.DecodeString(input[i+2 : i+4]); err == nil {
					sb.Write(b)
					i += 3 // skip 'x' and two hex digits; the loop will increment i past the last digit
					continue
				}
			}

			if shouldEscape(nextChar, inDoubleQuotes) {
				isEscaped = true
				continue
			}

			sb.WriteByte(currentChar)
			continue
		case '"':
			if !inSingleQuotes { // Toggle double quotes only if not in single quotes
				inDoubleQuotes = !inDoubleQuotes
			} else {
				sb.WriteByte(currentChar) // Treat as a regular character within single quotes
			}
		case '\'':
			if !inDoubleQuotes { // Toggle single quotes only if not in double quotes
				inSingleQuotes = !inSingleQuotes
			} else {
				sb.WriteByte(currentChar) // Treat as a regular character within double quotes
			}
		default:
			sb.WriteByte(currentChar)
		}
	}

	if inDoubleQuotes || inSingleQuotes {
		return "", fmt.Errorf("unquote `%q`: %w", input, ErrMismatchedQuotes)
	}

	if isEscaped {
		return "", fmt.Errorf("unquote `%q`: %w", input, ErrTrailingBackslash)
	}

	return sb.String(), nil
}

func shouldEscape(next byte, inDoubleQuotes bool) bool {
	switch next {
	case '\\', '"':
		return true
	case '\'':
		return !inDoubleQuotes
	case ' ', '\t', '\n':
		return true
	default:
		return false
	}
}
