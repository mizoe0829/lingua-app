export interface Language {
	code: string;
	name: string;
	native_name: string;
	flag: string;
	voice_name: string;
	pitch: number;
	rate: number;
}

export interface Token {
	id: number;
	text: string;
	clean_text: string;
	is_word: boolean;
	pos: string;
	ipa: string;
	meaning: string;
	liaison_hint: string;
}

export interface Sentence {
	id: number;
	raw_text: string;
	tokens: Token[];
}

export interface TokenizeResponse {
	lang_code: string;
	sentences: Sentence[];
	word_count: number;
}

export interface PresetText {
	id: string;
	lang_code: string;
	category: string;
	title: string;
	description: string;
	content: string;
	difficulty: 'Beginner' | 'Intermediate' | 'Advanced';
}

export interface Bookmark {
	id?: number;
	lang_code: string;
	text: string;
	ipa?: string;
	meaning?: string;
	context?: string;
	notes?: string;
	created_at?: string;
}

export type ProficiencyLevel = 'Beginner' | 'Intermediate' | 'Advanced';

export interface DialogMessage {
	id: string;
	role: 'user' | 'assistant';
	content: string;
	translation: string;
	better_phrasing?: string;
	pronunciation_tip?: string;
}

export interface DialogStreamChunk {
	type: 'reply' | 'translation' | 'better_phrasing' | 'tip' | 'done' | 'error';
	chunk?: string;
}

