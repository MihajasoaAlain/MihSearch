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

func IsStopWord(word string) bool {
	_, ok := stopWords[word]
	return ok
}

func RemoveStopWords(words []string) []string {
	result := make([]string, 0, len(words))
	for _, word := range words {
		if IsStopWord(word) {
			continue
		}
		result = append(result, word)
	}
	return result
}
