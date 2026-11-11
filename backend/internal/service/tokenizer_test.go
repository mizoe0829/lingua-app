package service

import (
	"testing"
)

func TestTokenizerService(t *testing.T) {
	// Initialize tokenizer with nil DB (pure heuristic fallback test)
	svc := NewTokenizerService(nil)

	t.Run("Empty text returns 0 sentences", func(t *testing.T) {
		res, err := svc.Tokenize("en-US", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Sentences) != 0 {
			t.Errorf("expected 0 sentences, got %d", len(res.Sentences))
		}
		if res.WordCount != 0 {
			t.Errorf("expected 0 words, got %d", res.WordCount)
		}
	})

	t.Run("Tokenizes multiple sentences with punctuation", func(t *testing.T) {
		text := "Hello world! How are you doing today? I'm fine."
		res, err := svc.Tokenize("en-US", text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(res.Sentences) != 3 {
			t.Errorf("expected 3 sentences, got %d", len(res.Sentences))
		}

		firstSentence := res.Sentences[0]
		if len(firstSentence.Tokens) == 0 {
			t.Fatalf("expected tokens in first sentence")
		}

		// First token should be "Hello"
		if firstSentence.Tokens[0].Text != "Hello" {
			t.Errorf("expected first token 'Hello', got '%s'", firstSentence.Tokens[0].Text)
		}
		if !firstSentence.Tokens[0].IsWord {
			t.Errorf("expected 'Hello' to be marked as word")
		}
	})

	t.Run("Detects linking consonant to vowel liaison", func(t *testing.T) {
		text := "Ask about it."
		res, err := svc.Tokenize("en-US", text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tokens := res.Sentences[0].Tokens
		// "about" ends with 't' (consonant) and "it" starts with 'i' (vowel) -> should trigger liaison hint
		var foundLiaison bool
		for _, tok := range tokens {
			if tok.CleanText == "about" && tok.LiaisonHint != "" {
				foundLiaison = true
				break
			}
		}

		if !foundLiaison {
			t.Errorf("expected liaison hint on 'about' before 'it'")
		}
	})
}

