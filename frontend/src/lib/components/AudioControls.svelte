<script lang="ts">
	import { speechEngine } from '$lib/audio.svelte';
	import { Play, Pause, Square, Gauge, Sliders } from 'lucide-svelte';

	interface Props {
		onStopAll?: () => void;
	}

	let { onStopAll }: Props = $props();

	const speeds = [0.75, 0.9, 1.0, 1.25];

	function handleStop() {
		speechEngine.stop();
		if (onStopAll) onStopAll();
	}
</script>

<div class="bg-slate-900/60 border border-slate-800/80 rounded-xl p-3.5 flex flex-wrap items-center justify-between gap-3 text-sm">
	<!-- Status indicator & Speed Selector -->
	<div class="flex items-center gap-3">
		<div class="flex items-center gap-2">
			<span class="relative flex h-2.5 w-2.5">
				{#if speechEngine.isPlaying}
					<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
					<span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500"></span>
				{:else}
					<span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-slate-600"></span>
				{/if}
			</span>
			<span class="text-xs font-semibold uppercase tracking-wider text-slate-400">
				{speechEngine.isPlaying ? '再生中 (音声合成)' : '待機中'}
			</span>
		</div>

		{#if speechEngine.isPlaying}
			<button
				onclick={handleStop}
				class="flex items-center gap-1 px-2.5 py-1 rounded bg-rose-500/20 hover:bg-rose-500/30 text-rose-300 border border-rose-500/30 text-xs font-medium transition-all"
			>
				<Square class="w-3 h-3 fill-current" /> 停止
			</button>
		{/if}
	</div>

	<!-- Speed controls -->
	<div class="flex items-center gap-2">
		<Gauge class="w-3.5 h-3.5 text-slate-400" />
		<span class="text-xs text-slate-400 font-medium">再生速度:</span>
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
			class="w-20 accent-indigo-500 h-1.5 bg-slate-800 rounded-lg cursor-pointer"
		/>
		<span class="text-xs font-mono text-slate-400">{speechEngine.pitch.toFixed(2)}</span>
	</div>
</div>

