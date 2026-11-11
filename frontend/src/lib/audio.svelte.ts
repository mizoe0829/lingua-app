/**
 * Svelte 5 Runes Audio State & Controller
 * Wraps Web Speech API with zero-overhead reactive state.
 */

export class SpeechEngine {
	isPlaying = $state(false);
	isPaused = $state(false);
	activeSentenceId = $state<number | null>(null);
	activeTokenId = $state<number | null>(null);
	rate = $state(0.95);
	pitch = $state(1.0);
	selectedLang = $state('en-US');

	private currentUtterance: SpeechSynthesisUtterance | null = null;
	private voices: SpeechSynthesisVoice[] = [];

	constructor() {
		if (typeof window !== 'undefined' && 'speechSynthesis' in window) {
			this.loadVoices();
			window.speechSynthesis.onvoiceschanged = () => {
				this.loadVoices();
			};
		}
	}

	private loadVoices() {
		if (typeof window === 'undefined') return;
		this.voices = window.speechSynthesis.getVoices();
	}

	private getVoiceForLang(langCode: string): SpeechSynthesisVoice | null {
		if (!this.voices.length) this.loadVoices();
		// Exact match
		let match = this.voices.find(v => v.lang.toLowerCase() === langCode.toLowerCase());
		if (match) return match;
		// Prefix match (e.g. "en" for "en-US")
		const prefix = langCode.split('-')[0].toLowerCase();
		match = this.voices.find(v => v.lang.toLowerCase().startsWith(prefix));
		return match || null;
	}

	/**
	 * Speak a single isolated word instantly (0ms latency)
	 */
	speakWord(word: string, langCode: string, tokenId?: number) {
		if (typeof window === 'undefined' || !('speechSynthesis' in window)) return;

		window.speechSynthesis.cancel();
		this.isPlaying = true;
		if (tokenId !== undefined) {
			this.activeTokenId = tokenId;
		}

		const utter = new SpeechSynthesisUtterance(word);
		utter.lang = langCode;
		utter.rate = this.rate;
		utter.pitch = this.pitch;

		const voice = this.getVoiceForLang(langCode);
		if (voice) utter.voice = voice;

		utter.onend = () => {
			this.isPlaying = false;
			if (tokenId !== undefined) {
				setTimeout(() => {
					if (this.activeTokenId === tokenId) this.activeTokenId = null;
				}, 400);
			}
		};

		utter.onerror = () => {
			this.isPlaying = false;
			this.activeTokenId = null;
		};

		window.speechSynthesis.speak(utter);
	}

	/**
	 * Speak a whole sentence with word boundary tracking for karaoke-style sync
	 */
	speakSentence(sentenceId: number, rawText: string, langCode: string, tokens: { id: number; text: string; is_word: boolean }[]) {
		if (typeof window === 'undefined' || !('speechSynthesis' in window)) return;

		window.speechSynthesis.cancel();
		this.isPlaying = true;
		this.isPaused = false;
		this.activeSentenceId = sentenceId;
		this.activeTokenId = null;

		const utter = new SpeechSynthesisUtterance(rawText);
		utter.lang = langCode;
		utter.rate = this.rate;
		utter.pitch = this.pitch;

		const voice = this.getVoiceForLang(langCode);
		if (voice) utter.voice = voice;

		// Word boundary tracking
		utter.onboundary = (e) => {
			if (e.name === 'word' && tokens.length > 0) {
				// Estimate matched token from character index
				const charIndex = e.charIndex;
				let accumulated = 0;
				for (const t of tokens) {
					const start = accumulated;
					const end = accumulated + t.text.length;
					if (charIndex >= start && charIndex <= end && t.is_word) {
						this.activeTokenId = t.id;
						break;
					}
					accumulated += t.text.length + 1; // approximate space
				}
			}
		};

		utter.onend = () => {
			this.isPlaying = false;
			this.isPaused = false;
			this.activeSentenceId = null;
			this.activeTokenId = null;
		};

		utter.onerror = () => {
			this.isPlaying = false;
			this.isPaused = false;
			this.activeSentenceId = null;
			this.activeTokenId = null;
		};

		this.currentUtterance = utter;
		window.speechSynthesis.speak(utter);
	}

	stop() {
		if (typeof window === 'undefined') return;
		window.speechSynthesis.cancel();
		this.isPlaying = false;
		this.isPaused = false;
		this.activeSentenceId = null;
		this.activeTokenId = null;
	}

	pause() {
		if (typeof window === 'undefined') return;
		window.speechSynthesis.pause();
		this.isPaused = true;
	}

	resume() {
		if (typeof window === 'undefined') return;
		window.speechSynthesis.resume();
		this.isPaused = false;
	}
}

export const speechEngine = new SpeechEngine();

