package service

import (
	"database/sql"
	"regexp"
	"strings"
	"unicode"

	"lingua-backend/internal/model"
)

type TokenizerService struct {
	db *sql.DB
}

func NewTokenizerService(db *sql.DB) *TokenizerService {
	return &TokenizerService{db: db}
}

// Tokenize parses input text into structured sentences and tokens with phonetic metadata
func (s *TokenizerService) Tokenize(langCode, text string) (*model.TokenizeResponse, error) {
	if strings.TrimSpace(text) == "" {
		return &model.TokenizeResponse{
			LangCode:  langCode,
			Sentences: []model.Sentence{},
			WordCount: 0,
		}, nil
	}

	// Split text into sentences using regex matching terminal punctuation (. ! ? etc.)
	sentenceSplitter := regexp.MustCompile(`(?m)([^.!?\n]+[.!?]*|\n+)`)
	rawMatches := sentenceSplitter.FindAllString(text, -1)

	var sentences []model.Sentence
	tokenId := 1
	totalWords := 0

	// Pre-load known vocabulary for this language into memory for O(1) matching
	vocabMap := make(map[string]model.VocabEntry)
	if s.db != nil {
		rows, err := s.db.Query(`SELECT clean_word, ipa, pos, meaning FROM vocabularies WHERE lang_code = ?`, langCode)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var v model.VocabEntry
				var clean string
				if err := rows.Scan(&clean, &v.IPA, &v.POS, &v.Meaning); err == nil {
					vocabMap[clean] = v
				}
			}
		}
	}

	for sIdx, raw := range rawMatches {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		// Tokenize words and punctuation while preserving original spacing
		tokens := s.splitSentenceToTokens(trimmed, &tokenId, vocabMap)
		for _, t := range tokens {
			if t.IsWord {
				totalWords++
			}
		}

		sentences = append(sentences, model.Sentence{
			ID:      sIdx + 1,
			RawText: trimmed,
			Tokens:  tokens,
		})
	}

	return &model.TokenizeResponse{
		LangCode:  langCode,
		Sentences: sentences,
		WordCount: totalWords,
	}, nil
}

// splitSentenceToTokens cuts text into words and punctuation
func (s *TokenizerService) splitSentenceToTokens(rawSentence string, nextId *int, vocabMap map[string]model.VocabEntry) []model.Token {
	var tokens []model.Token

	// Tokenize words, contractions (e.g. don't, I'm), and punctuations
	tokenRegex := regexp.MustCompile(`[\p{L}\p{M}]+(?:['’][\p{L}\p{M}]+)*|[0-9]+|[^\s\p{L}\p{M}0-9]+`)
	matches := tokenRegex.FindAllString(rawSentence, -1)

	for i, m := range matches {
		isWord := isAlphabetic(m)
		clean := strings.ToLower(strings.Trim(m, " .,!?;:\"'()[]{}«»¡¿"))

		token := model.Token{
			ID:        *nextId,
			Text:      m,
			CleanText: clean,
			IsWord:    isWord,
		}
		*nextId++

		if isWord && clean != "" {
			if vocab, found := vocabMap[clean]; found {
				token.IPA = vocab.IPA
				token.POS = vocab.POS
				token.Meaning = vocab.Meaning
			} else {
				// Provide fallback default classification
				token.POS = guessBasicPOS(clean)
			}

			// Liaison / linking phonetic hint detection
			if i+1 < len(matches) {
				nextMatch := matches[i+1]
				if isAlphabetic(nextMatch) {
					token.LiaisonHint = detectLiaison(m, nextMatch)
				}
			}
		}

		tokens = append(tokens, token)
	}

	return tokens
}

func isAlphabetic(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func guessBasicPOS(word string) string {
	switch {
	case strings.HasSuffix(word, "ly"):
		return "adverb"
	case strings.HasSuffix(word, "tion"), strings.HasSuffix(word, "ment"), strings.HasSuffix(word, "ness"):
		return "noun"
	case strings.HasSuffix(word, "able"), strings.HasSuffix(word, "ible"), strings.HasSuffix(word, "al"), strings.HasSuffix(word, "ive"):
		return "adjective"
	case strings.HasSuffix(word, "ed"), strings.HasSuffix(word, "ing"):
		return "verb"
	default:
		return "word"
	}
}

// detectLiaison provides phonetic linking hints (e.g., consonant followed by vowel in English/French)
func detectLiaison(current, next string) string {
	vowels := "aeiouyAEIOUY"
	if len(current) == 0 || len(next) == 0 {
		return ""
	}

	lastChar := rune(current[len(current)-1])
	firstCharNext := rune(next[0])

	// If ends with consonant and next starts with vowel: Linking consonant-to-vowel
	if !strings.ContainsRune(vowels, lastChar) && unicode.IsLetter(lastChar) && strings.ContainsRune(vowels, firstCharNext) {
		return "Linking consonant to vowel (滑らかに音を連結)"
	}

	return ""
}

