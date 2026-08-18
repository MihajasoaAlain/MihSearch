package analyzer

import (
	"reflect"
	"testing"
)

func TestAnalyzer(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "lowercases and strips punctuation",
			text: "Wireless Printer, with Wi-Fi!",
			want: []string{"wireless", "printer", "with", "wi", "fi"},
		},
		{
			name: "removes accents",
			text: "Imprimante à café déjà prête",
			want: []string{"imprimante", "a", "cafe", "deja", "prete"},
		},
		{
			name: "drops stop words",
			text: "le prix de la machine et des cartouches",
			want: []string{"prix", "machine", "cartouches"},
		},
		{
			name: "empty text yields no terms",
			text: "   ",
			want: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Analyzer(test.text)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Analyzer(%q) = %v, want %v", test.text, got, test.want)
			}
		})
	}
}

func TestAnalyzeTokensKeepsOriginalPositions(t *testing.T) {
	// "le" and "de" are dropped, but the surviving terms must keep the position
	// they held in the original stream, otherwise phrase search would drift.
	got := AnalyzeTokens("le prix de la machine")

	want := []Token{
		{Word: "prix", Position: 1},
		{Word: "machine", Position: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AnalyzeTokens() = %v, want %v", got, want)
	}
}
