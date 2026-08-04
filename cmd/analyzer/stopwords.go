package analyzer

var stopWords = map[string]struct{}{
	"le":  {},
	"la":  {},
	"les": {},
	"de":  {},
	"des": {},
	"du":  {},
	"un":  {},
	"une": {},
	"et":  {},
	"ou":  {},
}

func RemoveStopWords(words []string) []string {
	result := make([]string, 0, len(words))
	for _, word := range words {
		if _, ok := stopWords[word]; ok {
			continue
		}
		result = append(result, word)
	}
	return result
}
