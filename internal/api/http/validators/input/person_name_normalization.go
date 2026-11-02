package input

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func normalizePersonName(rawName string) string {
	return norm.NFC.String(strings.TrimSpace(rawName))
}
