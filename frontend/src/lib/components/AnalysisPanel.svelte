<script lang="ts">
	import type { Sentence, Token } from '$lib/types';
	import { Sparkles, X, Volume2, Loader2, Bot } from 'lucide-svelte';
	import { speechEngine } from '$lib/audio.svelte';

	interface Props {
		sentence: Sentence | null;
		focusedToken: Token | null;
		langCode: string;
		streamContent: string;
		isStreaming: boolean;
		onClose: () => void;
	}

	let { sentence, focusedToken, langCode, streamContent, isStreaming, onClose }: Props = $props();

	function handleSpeakTarget() {
		if (!sentence) return;
		if (focusedToken && focusedToken.is_word) {
			speechEngine.speakWord(focusedToken.clean_text, langCode, focusedToken.id);
		} else {
			speechEngine.speakSentence(sentence.id, sentence.raw_text, langCode, sentence.tokens);
		}
	}
</script>

{#if sentence}
	<div class="rounded-2xl border border-sky-500/30 bg-slate-900/90 backdrop-blur-md p-5 shadow-2xl shadow-sky-500/10 space-y-4">
		<!-- Header -->
		<div class="flex items-center justify-between pb-3 border-b border-slate-800">
			<div class="flex items-center gap-2">
				<div class="p-1.5 rounded-lg bg-sky-500/10 text-sky-400 border border-sky-500/20">
					<Bot class="w-4 h-4" />
				</div>
				<div>
					<h3 class="font-bold text-slate-100 flex items-center gap-2 text-sm sm:text-base">
						AI 音声・イントネーション & 構文解析
						{#if isStreaming}
							<span class="inline-flex items-center gap-1 text-xs text-sky-400 bg-sky-500/10 px-2 py-0.5 rounded-full border border-sky-500/20 animate-pulse">
								<Loader2 class="w-3 h-3 animate-spin" /> 解析ストリーミング中...
							</span>
						{/if}
					</h3>
					<p class="text-xs text-slate-400">Gemini 3.8 / 2.5 Flash + 高速ローカルフォールバック</p>
				</div>
			</div>

			<div class="flex items-center gap-2">
				<button
					onclick={handleSpeakTarget}
					class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/30 text-xs font-medium transition-all"
					title="対象フレーズを再生"
				>
					<Volume2 class="w-3.5 h-3.5" /> 再生
				</button>
				<button
					onclick={onClose}
					class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
					title="パネルを閉じる"
				>
					<X class="w-4 h-4" />
				</button>
			</div>
		</div>

		<!-- Target Context -->
		<div class="p-3 bg-slate-950/60 rounded-xl border border-slate-800/80 text-sm">
			<span class="text-xs font-semibold text-sky-400 uppercase tracking-wider block mb-1">分析対象:</span>
			<p class="text-slate-200 font-medium">"{sentence.raw_text}"</p>
			{#if focusedToken}
				<div class="mt-2 text-xs flex items-center gap-2 text-slate-400">
					<span>着目単語:</span>
					<span class="px-2 py-0.5 rounded bg-indigo-950 text-indigo-300 font-mono font-bold border border-indigo-800/40">
						{focusedToken.clean_text} {focusedToken.ipa}
					</span>
				</div>
			{/if}
		</div>

		<!-- Stream Content -->
		<div class="prose prose-invert prose-sm max-w-none text-slate-300 leading-relaxed max-h-96 overflow-y-auto pr-2 space-y-2">
			{#if streamContent}
				<div class="whitespace-pre-wrap font-sans text-sm">{streamContent}</div>
			{:else if isStreaming}
				<div class="flex items-center gap-2 text-slate-400 py-6 justify-center">
					<Loader2 class="w-5 h-5 animate-spin text-sky-400" />
					<span>音声音節と構文構造を分析中...</span>
				</div>
			{/if}
		</div>
	</div>
{/if}

