package analyzer

func Analyzer(text string) []string {
	text = Lowercase(text)
	text = RemovePunctuation(text)
	text = RemoveAccents(text)
	words := Tokenize(text)
	words = RemoveStopWords(words)
	return words
}
