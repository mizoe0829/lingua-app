package model

import "time"

// Language represents a supported language configuration
type Language struct {
	Code       string `json:"code"`        // e.g. "en-US", "es-ES", "fr-FR", "ja-JP"
	Name       string `json:"name"`        // e.g. "English (US)", "Español", "Français"
	NativeName string `json:"native_name"` // e.g. "English", "Español", "Français", "日本語"
	Flag       string `json:"flag"`        // e.g. "🇺🇸", "🇪🇸", "🇫🇷", "🇯🇵"
	VoiceName  string `json:"voice_name"`  // Default Web Speech synthesis voice identifier
	Pitch      float64 `json:"pitch"`       // Recommended pitch (1.0 default)
	Rate       float64 `json:"rate"`        // Recommended speaking rate (0.9 ~ 1.0)
}

// Token represents a single word or punctuation inside a sentence
type Token struct {
	ID          int     `json:"id"`
	Text        string  `json:"text"`         // e.g. "comfortable"
	CleanText   string  `json:"clean_text"`   // Lowercase, stripped punctuation
	IsWord      bool    `json:"is_word"`      // true if speakable word, false if punctuation/whitespace
	POS         string  `json:"pos"`          // Part of Speech: noun, verb, adj, adv, prep, etc.
	IPA         string  `json:"ipa"`          // International Phonetic Alphabet: e.g. "/ˈkʌmftəbl/"
	Meaning     string  `json:"meaning"`      // Japanese translation/definition
	LiaisonHint string  `json:"liaison_hint"` // Phonetic connection hint (e.g., "links with next vowel")
}

// Sentence represents a parsed sentence containing tokens
type Sentence struct {
	ID       int     `json:"id"`
	RawText  string  `json:"raw_text"`
	Tokens   []Token `json:"tokens"`
}

// TokenizeRequest is payload for /api/tokenize
type TokenizeRequest struct {
	LangCode string `json:"lang_code"`
	Text     string `json:"text"`
}

// TokenizeResponse is returned by /api/tokenize
type TokenizeResponse struct {
	LangCode  string     `json:"lang_code"`
	Sentences []Sentence `json:"sentences"`
	WordCount int        `json:"word_count"`
}

// PresetText represents sample content for quick start
type PresetText struct {
	ID          string `json:"id"`
	LangCode    string `json:"lang_code"`
	Category    string `json:"category"`    // e.g. "Daily Conversation", "Business", "Tech Presentation"
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Difficulty  string `json:"difficulty"`   // "Beginner", "Intermediate", "Advanced"
}

// VocabEntry represents vocabulary knowledge stored in SQLite
type VocabEntry struct {
	ID          int64     `json:"id"`
	LangCode    string    `json:"lang_code"`
	Word        string    `json:"word"`
	IPA         string    `json:"ipa"`
	POS         string    `json:"pos"`
	Meaning     string    `json:"meaning"`
	Example     string    `json:"example"`
	ExampleTranslation string `json:"example_translation"`
	CreatedAt   time.Time `json:"created_at"`
}

// Bookmark represents a saved word or phrase by the learner
type Bookmark struct {
	ID        int64     `json:"id"`
	LangCode  string    `json:"lang_code"`
	Text      string    `json:"text"`
	IPA       string    `json:"ipa"`
	Meaning   string    `json:"meaning"`
	Context   string    `json:"context"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// AnalyzeRequest represents the payload for AI phonetic & grammar streaming analysis
type AnalyzeRequest struct {
	LangCode string `json:"lang_code"`
	Sentence string `json:"sentence"`
	Word     string `json:"word,omitempty"` // Optional focused word
}

// DialogMessage represents a single message in the conversational sparring
type DialogMessage struct {
	Role             string `json:"role"`              // "user" or "assistant"
	Content          string `json:"content"`           // Foreign language utterance
	Translation      string `json:"translation"`       // Japanese translation
	BetterPhrasing   string `json:"better_phrasing"`   // Native alternative recommendation
	PronunciationTip string `json:"pronunciation_tip"` // Phonetic/grammar insight
}

// DialogStreamRequest is the payload for POST /api/dialog/stream
type DialogStreamRequest struct {
	LangCode    string          `json:"lang_code"`    // e.g. "en-US", "es-ES"
	Level       string          `json:"level"`        // "Beginner", "Intermediate", "Advanced"
	UserMessage string          `json:"user_message"` // Learner's current spoken utterance
	History     []DialogMessage `json:"history"`      // Conversation context
}

// DialogStreamChunk represents parsed SSE event payload
type DialogStreamChunk struct {
	Type  string `json:"type"`            // "reply", "translation", "better_phrasing", "tip", "done", "error"
	Chunk string `json:"chunk,omitempty"`
}

