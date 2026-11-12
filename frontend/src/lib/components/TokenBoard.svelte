<script lang="ts">
	import type { Sentence, Token } from '$lib/types';
	import { speechEngine } from '$lib/audio.svelte';
	import { Volume2, Sparkles, BookmarkPlus, Info, Activity } from 'lucide-svelte';
	import PitchVisualizer from './PitchVisualizer.svelte';

	interface Props {
		sentences: Sentence[];
		langCode: string;
		onAnalyzeSentence: (sentence: Sentence, token?: Token) => void;
		onBookmarkWord: (token: Token, sentenceText: string) => void;
	}

	let { sentences, langCode, onAnalyzeSentence, onBookmarkWord }: Props = $props();

	// Hovered token state for rich preview
	let hoveredToken = $state<Token | null>(null);
	let activePitchSentenceId = $state<number | null>(null);

	function handleWordClick(token: Token) {
		if (!token.is_word) return;
		speechEngine.speakWord(token.clean_text, langCode, token.id);
	}

	function handleSentencePlay(sentence: Sentence) {
		speechEngine.speakSentence(sentence.id, sentence.raw_text, langCode, sentence.tokens);
	}
</script>

<div class="space-y-6">
	{#each sentences as sentence (sentence.id)}
		<div
			class="group relative p-5 rounded-2xl border transition-all duration-200 {speechEngine.activeSentenceId === sentence.id ? 'border-indigo-500/60 bg-indigo-950/20 shadow-lg shadow-indigo-500/10' : 'border-slate-800 bg-slate-900/40 hover:border-slate-700 hover:bg-slate-900/70'}"
		>
			<!-- Sentence Header / Controls -->
			<div class="flex items-center justify-between pb-3 mb-3 border-b border-slate-800/60 text-xs text-slate-400">
				<div class="flex items-center gap-2">
					<span class="w-5 h-5 rounded-md bg-slate-800 flex items-center justify-center font-mono font-bold text-slate-300">
						{sentence.id}
					</span>
					<span class="text-slate-400 font-medium">文の単位 (Sense Group)</span>
				</div>

				<div class="flex items-center gap-2 opacity-90 group-hover:opacity-100 transition-opacity">
					<!-- Play Sentence Button -->
					<button
						onclick={() => handleSentencePlay(sentence)}
						class="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/30 font-medium text-xs transition-all hover:scale-[1.02]"
						title="文全体を音声再生 (Web Speech API)"
					>
						<Volume2 class="w-3.5 h-3.5" />
						<span>{speechEngine.activeSentenceId === sentence.id ? '再生中...' : '全文再生'}</span>
					</button>

					<!-- AI Analyze Button -->
					<button
						onclick={() => onAnalyzeSentence(sentence)}
						class="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-sky-500/10 hover:bg-sky-500/20 text-sky-300 border border-sky-500/30 font-medium text-xs transition-all hover:scale-[1.02]"
						title="Gemini AIによる発音・イントネーション・構文ストリーミング解説"
					>
						<Sparkles class="w-3.5 h-3.5 text-sky-400" />
						<span>AI発音・構文解説</span>
					</button>

					<!-- Pitch Contour Button -->
					<button
						onclick={() => (activePitchSentenceId = activePitchSentenceId === sentence.id ? null : sentence.id)}
						class="flex items-center gap-1.5 px-3 py-1 rounded-lg transition-all hover:scale-[1.02] text-xs font-medium {activePitchSentenceId === sentence.id ? 'bg-amber-500/25 text-amber-300 border border-amber-500/40 shadow-sm shadow-amber-500/20' : 'bg-amber-500/10 hover:bg-amber-500/20 text-amber-300/80 hover:text-amber-200 border border-amber-500/20'}"
						title="ネイティブのお手本イントネーション曲線と、自分のマイク発声ピッチをリアルタイム比較"
					>
						<Activity class="w-3.5 h-3.5 text-amber-400" />
						<span>ピッチ波形比較</span>
					</button>
				</div>
			</div>

			<!-- Interactive Tokens Flow -->
			<div class="flex flex-wrap items-baseline gap-x-1.5 gap-y-2.5 text-lg sm:text-xl leading-relaxed">
				{#each sentence.tokens as token (token.id)}
					{#if token.is_word}
						<div class="relative group/token inline-block">
							<button
								type="button"
								onclick={() => handleWordClick(token)}
								onmouseenter={() => (hoveredToken = token)}
								onmouseleave={() => (hoveredToken = null)}
								class="px-1.5 py-0.5 rounded-lg font-medium transition-all duration-150 cursor-pointer select-none text-left
									{speechEngine.activeTokenId === token.id
										? 'bg-amber-400 text-slate-950 scale-105 font-bold shadow-md shadow-amber-400/30 ring-2 ring-amber-300'
										: token.ipa
										? 'hover:bg-indigo-600/30 text-slate-100 hover:text-white border-b-2 border-indigo-400/50'
										: 'hover:bg-slate-800 text-slate-200 hover:text-white'}"
							>
								{token.text}
							</button>

							<!-- Liaison Hint Badge (if detected) -->
							{#if token.liaison_hint}
								<span
									class="inline-block text-[10px] text-sky-400 font-mono font-bold tracking-tighter ml-0.5 cursor-help"
									title={token.liaison_hint}
								>
									⁀
								</span>
							{/if}

							<!-- Hover Tooltip Popup for Phonetics & Meaning -->
							<div
								class="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 hidden group-hover/token:flex flex-col z-40 w-56 p-2.5 bg-slate-900/95 backdrop-blur-md rounded-xl border border-indigo-500/30 shadow-xl shadow-black/50 text-xs pointer-events-auto transition-all animate-in fade-in zoom-in-95"
							>
								<div class="flex items-center justify-between pb-1.5 border-b border-slate-800">
									<div class="flex items-center gap-1.5">
										<span class="font-bold text-white text-sm">{token.clean_text}</span>
										{#if token.ipa}
											<span class="font-mono text-indigo-300 bg-indigo-950/60 px-1 rounded text-[11px]">{token.ipa}</span>
										{/if}
									</div>
									<button
										onclick={(e) => {
											e.stopPropagation();
											onBookmarkWord(token, sentence.raw_text);
										}}
										class="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-amber-400 transition-colors"
										title="単語帳に追加"
									>
										<BookmarkPlus class="w-3.5 h-3.5" />
									</button>
								</div>

								<div class="pt-1.5 space-y-1">
									{#if token.pos}
										<div class="flex items-center gap-1 text-[11px] text-slate-400">
											<span class="px-1 py-0.2 bg-slate-800 rounded uppercase font-semibold text-[9px] text-slate-300">{token.pos}</span>
											{#if token.meaning}
												<span class="text-slate-200">{token.meaning}</span>
											{/if}
										</div>
									{/if}

									{#if token.liaison_hint}
										<div class="text-[10px] text-sky-400 bg-sky-950/40 p-1 rounded border border-sky-800/40">
											🔗 {token.liaison_hint}
										</div>
									{/if}

									<div class="pt-1 flex items-center justify-between text-[10px] text-slate-400 border-t border-slate-800/50">
										<span class="text-indigo-400">クリックで即時発音</span>
										<button
											onclick={(e) => {
												e.stopPropagation();
												onAnalyzeSentence(sentence, token);
											}}
											class="text-sky-400 hover:underline"
										>
											AI解説 →
										</button>
									</div>
								</div>

								<!-- Arrow indicator -->
								<div class="absolute left-1/2 -translate-x-1/2 top-full w-2 h-2 bg-slate-900 border-r border-b border-indigo-500/30 rotate-45 -mt-1"></div>
							</div>
						</div>
					{:else}
						<!-- Punctuation or whitespace -->
						<span class="text-slate-400 select-none">{token.text}</span>
					{/if}
				{/each}
			</div>

			<!-- Pitch Contour Visualizer for this sentence -->
			{#if activePitchSentenceId === sentence.id}
				<PitchVisualizer
					{sentence}
					{langCode}
					onClose={() => (activePitchSentenceId = null)}
				/>
			{/if}
		</div>
	{/each}
</div>

