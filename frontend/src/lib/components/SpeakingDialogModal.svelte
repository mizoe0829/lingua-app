<script lang="ts">
	import type { DialogMessage, ProficiencyLevel, Language } from '$lib/types';
	import { speechEngine } from '$lib/audio.svelte';
	import { X, Mic, Send, Volume2, Sparkles, MessageSquare, Bot, User, Gauge, Lightbulb, CheckCircle2, RotateCcw } from 'lucide-svelte';

	interface Props {
		isOpen: boolean;
		selectedLang: string;
		languages: Language[];
		onClose: () => void;
	}

	let { isOpen, selectedLang, languages, onClose }: Props = $props();

	// State
	let level = $state<ProficiencyLevel>('Intermediate');
	let messages = $state<DialogMessage[]>([]);
	let userInput = $state('');
	let isStreaming = $state(false);
	let autoSpeak = $state(true);
	let abortController: AbortController | null = null;

	let currentLangObj = $derived(
		languages.find(l => l.code === selectedLang) || { name: 'English', flag: '🇺🇸' }
	);

	// Recommended starter prompts per language
	const starters: Record<string, string[]> = {
		'en-US': [
			"Could you recommend a good café around here?",
			"I'm working on refactoring a legacy Go service today.",
			"What do you think is the best way to improve speaking fluency?"
		],
		'es-ES': [
			"¿Me puedes recomendar un buen lugar para comer tapas?",
			"Estoy aprendiendo español para viajar a Madrid.",
			"¿Cuál es la diferencia entre 'por' y 'para'?"
		],
		'fr-FR': [
			"Pourriez-vous me conseiller un bon musée à visiter ?",
			"J'adore le café et les croissants le matin.",
			"Comment puis-je améliorer ma prononciation en français ?"
		],
		'de-DE': [
			"Können Sie mir ein gutes Restaurant in der Nähe empfehlen?",
			"Ich lerne Deutsch, weil ich die Kultur sehr mag.",
			"Wie sagt man 'software architecture' auf Deutsch?"
		],
		'it-IT': [
			"Mi puoi consigliare una buona pizzeria qui vicino?",
			"Vorrei ordinare un cappuccino e un cornetto.",
			"Qual è il modo migliore per praticare l'italiano ogni giorno?"
		]
	};

	let currentStarters = $derived(starters[selectedLang] || starters['en-US']);

	async function handleSendMessage(textToSend?: string) {
		const text = (textToSend || userInput).trim();
		if (!text || isStreaming) return;

		const userMsg: DialogMessage = {
			id: 'user-' + Date.now(),
			role: 'user',
			content: text,
			translation: '' // will be contextualized
		};

		messages = [...messages, userMsg];
		userInput = '';
		isStreaming = true;

		// Initialize assistant reply holder
		const assistantMsgId = 'assistant-' + Date.now();
		const assistantMsg: DialogMessage = {
			id: assistantMsgId,
			role: 'assistant',
			content: '',
			translation: '',
			better_phrasing: '',
			pronunciation_tip: ''
		};
		messages = [...messages, assistantMsg];

		if (abortController) abortController.abort();
		abortController = new AbortController();

		try {
			const res = await fetch('/api/dialog/stream', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					lang_code: selectedLang,
					level: level,
					user_message: text,
					history: messages.slice(0, -2) // pass previous context
				}),
				signal: abortController.signal
			});

			if (!res.ok) {
				const errMsg = await res.text();
				updateAssistantMessage(assistantMsgId, { content: 'エラーが発生しました: ' + errMsg });
				isStreaming = false;
				return;
			}

			const reader = res.body?.getReader();
			if (!reader) return;

			const decoder = new TextDecoder();
			let buffer = '';

			while (true) {
				const { value, done } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() || '';

				for (const line of lines) {
					const trimmed = line.trim();
					if (trimmed.startsWith('data:')) {
						const jsonStr = trimmed.substring(5).trim();
						if (jsonStr) {
							try {
								const parsed = JSON.parse(jsonStr);
								if (parsed.type === 'reply' && parsed.chunk) {
									appendToAssistantMessage(assistantMsgId, 'content', parsed.chunk);
								} else if (parsed.type === 'translation' && parsed.chunk) {
									appendToAssistantMessage(assistantMsgId, 'translation', parsed.chunk);
								} else if (parsed.type === 'better_phrasing' && parsed.chunk) {
									appendToAssistantMessage(assistantMsgId, 'better_phrasing', parsed.chunk);
								} else if (parsed.type === 'tip' && parsed.chunk) {
									appendToAssistantMessage(assistantMsgId, 'pronunciation_tip', parsed.chunk);
								} else if (parsed.type === 'done') {
									isStreaming = false;
									// Auto-speak assistant response if enabled
									const finalMsg = messages.find(m => m.id === assistantMsgId);
									if (finalMsg && autoSpeak && finalMsg.content.trim()) {
										// Adjust speech rate by level
										const rateByLevel = level === 'Beginner' ? 0.85 : level === 'Advanced' ? 1.1 : 0.95;
										speechEngine.rate = rateByLevel;
										speechEngine.speakWord(finalMsg.content, selectedLang);
									}
								}
							} catch {
								// Ignore parse chunk
							}
						}
					}
				}
			}
		} catch (err: any) {
			if (err.name !== 'AbortError') {
				updateAssistantMessage(assistantMsgId, { content: '[通信が切断されました]' });
			}
		} finally {
			isStreaming = false;
		}
	}

	function appendToAssistantMessage(id: string, field: 'content' | 'translation' | 'better_phrasing' | 'pronunciation_tip', chunk: string) {
		messages = messages.map(m => {
			if (m.id === id) {
				return { ...m, [field]: (m[field] || '') + chunk };
			}
			return m;
		});
	}

	function updateAssistantMessage(id: string, updates: Partial<DialogMessage>) {
		messages = messages.map(m => (m.id === id ? { ...m, ...updates } : m));
	}

	function handleSpeakText(text: string) {
		speechEngine.speakWord(text, selectedLang);
	}

	function handleClearChat() {
		messages = [];
		if (abortController) abortController.abort();
		isStreaming = false;
	}
</script>

{#if isOpen}
	<!-- Backdrop -->
	<div
		tabindex="0"
		role="button"
		class="fixed inset-0 z-50 bg-black/75 backdrop-blur-md flex items-center justify-center p-3 sm:p-6 animate-in fade-in"
		onclick={onClose}
		onkeydown={(e) => e.key === 'Escape' && onClose()}
	>
		<!-- Modal Box -->
		<div
			tabindex="-1"
			role="dialog"
			aria-modal="true"
			class="relative w-full max-w-4xl h-[90vh] bg-slate-900 border border-slate-800 rounded-3xl shadow-2xl flex flex-col overflow-hidden"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<!-- Top Header -->
			<div class="px-6 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/60">
				<div class="flex items-center gap-3">
					<div class="p-2.5 rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-600 text-white shadow-lg shadow-sky-500/20">
						<MessageSquare class="w-5 h-5" />
					</div>
					<div>
						<div class="flex items-center gap-2">
							<h3 class="font-extrabold text-white text-base sm:text-lg">
								AIスピーキング壁打ちバディ
							</h3>
							<span class="text-xs px-2.5 py-0.5 rounded-full font-semibold bg-sky-500/10 text-sky-400 border border-sky-500/20">
								{currentLangObj.flag} {currentLangObj.name}
							</span>
						</div>
						<p class="text-xs text-slate-400 hidden sm:block">
							話しかけると即座に返答・日本語訳・ネイティブ言い換えをフィードバック
						</p>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<!-- Auto Speak Toggle -->
					<button
						onclick={() => (autoSpeak = !autoSpeak)}
						class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl border text-xs font-semibold transition-all {autoSpeak ? 'bg-indigo-600/20 border-indigo-500/40 text-indigo-300' : 'bg-slate-950 border-slate-800 text-slate-500'}"
						title="返答を自動で音声再生"
					>
						<Volume2 class="w-3.5 h-3.5" />
						<span class="hidden sm:inline">音声自動読み上げ</span>
					</button>

					<!-- Clear Chat -->
					{#if messages.length > 0}
						<button
							onclick={handleClearChat}
							class="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
							title="会話履歴をクリア"
						>
							<RotateCcw class="w-4 h-4" />
						</button>
					{/if}

					<!-- Close -->
					<button
						onclick={onClose}
						class="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
					>
						<X class="w-5 h-5" />
					</button>
				</div>
			</div>

			<!-- Level Bar -->
			<div class="px-6 py-2.5 bg-slate-950/40 border-b border-slate-800/80 flex items-center justify-between text-xs">
				<div class="flex items-center gap-2">
					<Gauge class="w-4 h-4 text-slate-400" />
					<span class="font-semibold text-slate-300">習熟度レベル:</span>
				</div>

				<div class="flex items-center gap-1.5">
					<button
						onclick={() => (level = 'Beginner')}
						class="px-3 py-1 rounded-lg text-xs font-bold transition-all {level === 'Beginner' ? 'bg-emerald-500 text-slate-950 shadow-md shadow-emerald-500/20' : 'bg-slate-800 text-slate-400 hover:text-white'}"
					>
						🟢 初級 (Beginner)
					</button>
					<button
						onclick={() => (level = 'Intermediate')}
						class="px-3 py-1 rounded-lg text-xs font-bold transition-all {level === 'Intermediate' ? 'bg-amber-400 text-slate-950 shadow-md shadow-amber-400/20' : 'bg-slate-800 text-slate-400 hover:text-white'}"
					>
						🟡 中級 (Intermediate)
					</button>
					<button
						onclick={() => (level = 'Advanced')}
						class="px-3 py-1 rounded-lg text-xs font-bold transition-all {level === 'Advanced' ? 'bg-rose-500 text-white shadow-md shadow-rose-500/20' : 'bg-slate-800 text-slate-400 hover:text-white'}"
					>
						🔴 上級 (Advanced)
					</button>
				</div>
			</div>

			<!-- Message List Area -->
			<div class="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
				{#if messages.length === 0}
					<div class="h-full flex flex-col items-center justify-center text-center max-w-md mx-auto space-y-4 py-8">
						<div class="w-14 h-14 rounded-3xl bg-indigo-600/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
							<Bot class="w-7 h-7" />
						</div>
						<div class="space-y-1">
							<h4 class="font-bold text-slate-200 text-base">どんなことでも話しかけてください</h4>
							<p class="text-xs text-slate-400 leading-relaxed">
								AIが選んだ習熟度レベルに合わせて対話し、会話の返答・日本語訳・より自然なネイティブ表現を即座に教えてくれます。
							</p>
						</div>

						<!-- Starter Chips -->
						<div class="w-full text-left space-y-2 pt-2">
							<span class="text-[11px] font-bold text-slate-500 uppercase tracking-wider block">話題のきっかけ（クリックで送信）:</span>
							<div class="flex flex-col gap-1.5">
								{#each currentStarters as st}
									<button
										onclick={() => handleSendMessage(st)}
										class="p-2.5 rounded-xl bg-slate-950/70 border border-slate-800 hover:border-indigo-500/50 hover:bg-slate-800 text-xs text-slate-300 hover:text-white text-left transition-all flex items-center justify-between group"
									>
										<span>"{st}"</span>
										<span class="text-indigo-400 opacity-0 group-hover:opacity-100 transition-opacity">→</span>
									</button>
								{/each}
							</div>
						</div>
					</div>
				{:else}
					{#each messages as msg (msg.id)}
						{#if msg.role === 'user'}
							<!-- User Message -->
							<div class="flex justify-end items-start gap-3">
								<div class="max-w-[85%] sm:max-w-[70%] space-y-1">
									<div class="p-4 rounded-2xl rounded-tr-none bg-indigo-600 text-white shadow-lg shadow-indigo-600/10">
										<p class="text-sm sm:text-base font-medium leading-relaxed">{msg.content}</p>
									</div>
									<div class="flex items-center justify-end gap-1.5 text-[11px] text-slate-400">
										<button onclick={() => handleSpeakText(msg.content)} class="hover:text-indigo-300">
											<Volume2 class="w-3.5 h-3.5 inline" /> 発音
										</button>
									</div>
								</div>
								<div class="w-8 h-8 rounded-full bg-indigo-500 flex items-center justify-center text-white shrink-0 shadow-md">
									<User class="w-4 h-4" />
								</div>
							</div>
						{:else}
							<!-- Assistant Message -->
							<div class="flex justify-start items-start gap-3">
								<div class="w-8 h-8 rounded-full bg-gradient-to-tr from-sky-500 to-indigo-600 flex items-center justify-center text-white shrink-0 shadow-md shadow-sky-500/20">
									<Bot class="w-4 h-4" />
								</div>

								<div class="max-w-[90%] sm:max-w-[80%] space-y-2.5">
									<!-- Foreign Reply -->
									<div class="p-4 rounded-2xl rounded-tl-none bg-slate-950 border border-slate-800 space-y-2 shadow-xl">
										<div class="flex items-center justify-between border-b border-slate-800/60 pb-1.5">
											<span class="text-[11px] font-bold text-sky-400 uppercase tracking-wider">AI Response</span>
											<button
												onclick={() => handleSpeakText(msg.content)}
												class="p-1 rounded text-slate-400 hover:text-sky-300 transition-colors"
												title="音声を再生"
											>
												<Volume2 class="w-4 h-4" />
											</button>
										</div>
										<p class="text-sm sm:text-base text-slate-100 font-medium leading-relaxed">
											{msg.content || '思考中...'}
										</p>

										<!-- Japanese Translation -->
										{#if msg.translation}
											<div class="pt-2 border-t border-slate-800/60 text-xs text-slate-400 leading-relaxed">
												<span class="font-bold text-slate-500 mr-1">日本語訳:</span>
												<span>{msg.translation}</span>
											</div>
										{/if}
									</div>

									<!-- Better Phrasing & Tips (Learning Value) -->
									{#if msg.better_phrasing || msg.pronunciation_tip}
										<div class="p-3.5 rounded-xl bg-amber-400/5 border border-amber-400/20 space-y-2 text-xs">
											{#if msg.better_phrasing}
												<div>
													<span class="font-bold text-amber-400 flex items-center gap-1 mb-0.5">
														<Sparkles class="w-3.5 h-3.5" /> ネイティブの自然な言い換え (Better Phrasing):
													</span>
													<p class="text-slate-300 leading-relaxed font-sans">{msg.better_phrasing}</p>
												</div>
											{/if}

											{#if msg.pronunciation_tip}
												<div class="pt-1.5 border-t border-amber-400/10">
													<span class="font-bold text-amber-300/80 flex items-center gap-1 mb-0.5">
														<Lightbulb class="w-3.5 h-3.5" /> 発音・イントネーションのコツ:
													</span>
													<p class="text-slate-400 leading-relaxed">{msg.pronunciation_tip}</p>
												</div>
											{/if}
										</div>
									{/if}
								</div>
							</div>
						{/if}
					{/each}
				{/if}
			</div>

			<!-- Input Area -->
			<div class="p-4 bg-slate-950/80 border-t border-slate-800 space-y-2">
				<form onsubmit={(e) => { e.preventDefault(); handleSendMessage(); }} class="flex items-center gap-2">
					<input
						type="text"
						bind:value={userInput}
						disabled={isStreaming}
						placeholder="{currentLangObj.name} で話しかけてください..."
						class="flex-1 bg-slate-900 border border-slate-800 rounded-2xl px-4 py-3 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:opacity-50"
					/>

					<button
						type="submit"
						disabled={isStreaming || !userInput.trim()}
						class="p-3 rounded-2xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold shadow-lg shadow-indigo-600/30 transition-all hover:scale-105 active:scale-95 disabled:opacity-40"
						title="送信"
					>
						<Send class="w-4 h-4" />
					</button>
				</form>
				<p class="text-[11px] text-center text-slate-500">
					※ 次のステップでマイク長押しによるリアルタイム音声認識（STT）とピッチ波形比較が追加されます。
				</p>
			</div>
		</div>
	</div>
{/if}

