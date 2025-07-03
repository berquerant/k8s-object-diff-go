package internal

import "al.essio.dev/pkg/shellescape"

func EscapeCommand(command string) string {
	return shellescape.Quote(command)
}
