package analyzer

// Token is a normalized term together with the position it held in the original
// token stream. Positions are kept across stop-word removal so that phrase and
// proximity queries stay possible later on.
type Token struct {
	Word     string
	Position int
}

// Analyzer normalizes text and returns the terms that survive the pipeline.
func Analyzer(text string) []string {
	tokens := AnalyzeTokens(text)
	words := make([]string, 0, len(tokens))
	for _, token := range tokens {
		words = append(words, token.Word)
	}
	return words
}

// AnalyzeTokens runs the full pipeline and keeps each term's original position.
func AnalyzeTokens(text string) []Token {
	text = Lowercase(text)
	text = RemovePunctuation(text)
	text = RemoveAccents(text)
	words := Tokenize(text)

	tokens := make([]Token, 0, len(words))
	for position, word := range words {
		if IsStopWord(word) {
			continue
		}
		tokens = append(tokens, Token{Word: word, Position: position})
	}
	return tokens
}
