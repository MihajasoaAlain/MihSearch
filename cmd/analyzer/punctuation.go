package analyzer

import "regexp"

var punctuation = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func RemovePunctuation(text string) string {
	return punctuation.ReplaceAllString(text, " ")
}
