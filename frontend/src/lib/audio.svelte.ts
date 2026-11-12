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

/**
 * Svelte 5 Runes Speech-to-Text (STT) Controller
 * Uses Web Speech API SpeechRecognition / webkitSpeechRecognition
 */
export class SpeechRecognizer {
	isListening = $state(false);
	interimTranscript = $state('');
	finalTranscript = $state('');
	errorMessage = $state<string | null>(null);
	isSupported = $state(false);

	private recognition: any = null;
	private onResultCallback: ((text: string, isFinal: boolean) => void) | null = null;

	constructor() {
		if (typeof window !== 'undefined') {
			const SpeechRec = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
			this.isSupported = !!SpeechRec;
		}
	}

	start(langCode: string, onResult?: (text: string, isFinal: boolean) => void) {
		if (typeof window === 'undefined') return;

		const SpeechRec = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
		if (!SpeechRec) {
			this.errorMessage = 'お使いのブラウザは音声認識に対応していません（Chrome/Edge推奨）';
			return;
		}

		if (this.isListening) {
			this.stop();
			return;
		}

		this.errorMessage = null;
		this.interimTranscript = '';
		this.finalTranscript = '';
		this.onResultCallback = onResult || null;

		try {
			this.recognition = new SpeechRec();
			this.recognition.lang = langCode;
			this.recognition.continuous = true;
			this.recognition.interimResults = true;
			this.recognition.maxAlternatives = 1;

			this.recognition.onstart = () => {
				this.isListening = true;
			};

			this.recognition.onresult = (event: any) => {
				let interim = '';
				let final = '';

				for (let i = event.resultIndex; i < event.results.length; ++i) {
					const text = event.results[i][0].transcript;
					if (event.results[i].isFinal) {
						final += text;
					} else {
						interim += text;
					}
				}

				if (final) {
					this.finalTranscript += final;
					if (this.onResultCallback) {
						this.onResultCallback(this.finalTranscript, true);
					}
				}
				this.interimTranscript = interim;
				if (interim && this.onResultCallback) {
					this.onResultCallback(this.finalTranscript + ' ' + interim, false);
				}
			};

			this.recognition.onerror = (event: any) => {
				console.warn('[SpeechRecognizer] error:', event.error);
				if (event.error === 'not-allowed') {
					this.errorMessage = 'マイクの使用が許可されていません。ブラウザの設定でマイクを許可してください。';
				} else if (event.error === 'no-speech') {
					// No speech detected - benign
				} else {
					this.errorMessage = `音声認識エラー: ${event.error}`;
				}
				this.isListening = false;
			};

			this.recognition.onend = () => {
				this.isListening = false;
			};

			this.recognition.start();
		} catch (err: any) {
			console.error('[SpeechRecognizer] start failed:', err);
			this.errorMessage = 'マイクの起動に失敗しました';
			this.isListening = false;
		}
	}

	stop() {
		if (this.recognition) {
			try {
				this.recognition.stop();
			} catch {
				// Ignore
			}
			this.recognition = null;
		}
		this.isListening = false;
	}
}

export const speechRecognizer = new SpeechRecognizer();

