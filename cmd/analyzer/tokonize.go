package analyzer

import "strings"

func Tokenize(text string) []string {
	return strings.Fields(text)
}
