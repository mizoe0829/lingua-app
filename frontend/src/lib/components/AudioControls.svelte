<script lang="ts">
	import type { Sentence } from '$lib/types';
	import { speechEngine } from '$lib/audio.svelte';
	import { Play, Pause, Square, Gauge, Sliders, Volume2 } from 'lucide-svelte';

	interface Props {
		sentences?: Sentence[];
		langCode?: string;
		onStopAll?: () => void;
	}

	let { sentences = [], langCode = 'en-US', onStopAll }: Props = $props();

	const speeds = [0.75, 0.9, 1.0, 1.25];

	function handleTogglePlayAll() {
		if (speechEngine.isPlaying) {
			speechEngine.stop();
			if (onStopAll) onStopAll();
		} else {
			speechEngine.playAll(sentences, langCode);
		}
	}
</script>

<div class="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-4 flex flex-wrap items-center justify-between gap-4 text-sm shadow-xl">
	<!-- Left: Play All Button & Status -->
	<div class="flex items-center gap-3">
		<!-- Main Play All Button -->
		<button
			onclick={handleTogglePlayAll}
			disabled={sentences.length === 0}
			class="flex items-center gap-2 px-4 py-2 rounded-xl font-bold text-xs sm:text-sm transition-all hover:scale-105 active:scale-95 disabled:opacity-40 shadow-lg {speechEngine.isPlaylistPlaying ? 'bg-amber-500 hover:bg-amber-400 text-slate-950 shadow-amber-500/30' : 'bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-400 hover:to-teal-500 text-white shadow-emerald-500/30'}"
			title="登録されたすべての文を順番にノンストップでカラオケ風ハイライト再生"
		>
			{#if speechEngine.isPlaylistPlaying}
				<Square class="w-4 h-4 fill-current animate-pulse" />
				<span>再生停止 (文 {speechEngine.playlistIndex + 1}/{sentences.length})</span>
			{:else}
				<Play class="w-4 h-4 fill-current" />
				<span>全文を一括再生 ({sentences.length}文)</span>
			{/if}
		</button>

		<!-- Status indicator -->
		<div class="hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-lg bg-slate-950 border border-slate-800">
			<span class="relative flex h-2 w-2">
				{#if speechEngine.isPlaying}
					<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
					<span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
				{:else}
					<span class="relative inline-flex rounded-full h-2 w-2 bg-slate-600"></span>
				{/if}
			</span>
			<span class="text-[11px] font-semibold text-slate-400">
				{speechEngine.isPlaylistPlaying ? '一括連続再生中' : speechEngine.isPlaying ? '単体発話中' : '待機中'}
			</span>
		</div>

		{#if speechEngine.isPlaying && !speechEngine.isPlaylistPlaying}
			<button
				onclick={() => speechEngine.stop()}
				class="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 border border-rose-500/30 text-xs font-medium transition-all"
			>
				<Square class="w-3 h-3 fill-current" /> 停止
			</button>
		{/if}
	</div>

	<!-- Right: Speed & Pitch adjustments -->
	<div class="flex items-center gap-4 flex-wrap">
		<!-- Speed controls -->
		<div class="flex items-center gap-2">
			<Gauge class="w-3.5 h-3.5 text-slate-400" />
			<span class="text-xs text-slate-400 font-medium">速度:</span>
			<div class="flex items-center gap-1 bg-slate-950 p-1 rounded-lg border border-slate-800">
				{#each speeds as s}
					<button
						onclick={() => speechEngine.rate = s}
						class="px-2 py-0.5 rounded text-xs font-medium transition-colors {speechEngine.rate === s ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
					>
						{s}x
					</button>
				{/each}
			</div>
		</div>

		<!-- Pitch adjustments -->
		<div class="flex items-center gap-2">
			<Sliders class="w-3.5 h-3.5 text-slate-400" />
			<span class="text-xs text-slate-400 font-medium">ピッチ:</span>
			<input
				type="range"
				min="0.8"
				max="1.2"
				step="0.05"
				bind:value={speechEngine.pitch}
				class="w-16 sm:w-20 accent-indigo-500 h-1.5 bg-slate-800 rounded-lg cursor-pointer"
			/>
			<span class="text-xs font-mono text-slate-400">{speechEngine.pitch.toFixed(2)}</span>
		</div>
	</div>
</div>

