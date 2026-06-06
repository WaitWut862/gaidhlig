package gla

import (
	"fmt"
	"testing"
)

func TestAnalyse(t *testing.T) {
	sentences, err := Analyse("Tha mi a' dol dhan bhùth an-diugh.")
	if err != nil {
		t.Fatalf("Analyse error: %v", err)
	}

	if len(sentences) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(sentences))
	}

	s := sentences[0]
	fmt.Printf("Sentence: %s\n", s.Text)
	for _, tok := range s.Tokens {
		fmt.Printf("  %d\t%s\t%s\t%s\t%s\n", tok.ID, tok.Form, tok.Lemma, tok.UPOS, tok.DepRel)
	}

	if len(s.Tokens) != 8 {
		t.Errorf("expected 8 tokens, got %d", len(s.Tokens))
	}

	// Verify a few key tokens
	if s.Tokens[0].Lemma != "bi" {
		t.Errorf("expected lemma 'bi' for Tha, got %q", s.Tokens[0].Lemma)
	}
	if s.Tokens[5].Lemma != "bùth" {
		t.Errorf("expected lemma 'bùth' for bhùth, got %q", s.Tokens[5].Lemma)
	}
}
