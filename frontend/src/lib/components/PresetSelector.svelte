<script lang="ts">
	import type { PresetText } from '$lib/types';
	import { BookOpen, Sparkles, Flame } from 'lucide-svelte';

	interface Props {
		presets: PresetText[];
		onSelectPreset: (preset: PresetText) => void;
	}

	let { presets, onSelectPreset }: Props = $props();
</script>

{#if presets.length > 0}
	<div class="space-y-3">
		<div class="flex items-center justify-between">
			<h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
				<BookOpen class="w-3.5 h-3.5 text-indigo-400" />
				学習用プリセット文 (ワンタップで学習開始)
			</h3>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
			{#each presets as preset (preset.id)}
				<button
					type="button"
					onclick={() => onSelectPreset(preset)}
					class="text-left p-3.5 rounded-xl bg-slate-900/60 border border-slate-800 hover:border-indigo-500/50 hover:bg-slate-900 transition-all duration-200 group flex flex-col justify-between"
				>
					<div>
						<div class="flex items-center justify-between mb-1.5">
							<span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700">
								{preset.category}
							</span>
							<span class="text-[10px] font-semibold text-indigo-400">
								{preset.difficulty}
							</span>
						</div>
						<h4 class="font-bold text-sm text-slate-200 group-hover:text-white transition-colors mb-1 line-clamp-1">
							{preset.title}
						</h4>
						<p class="text-xs text-slate-400 line-clamp-2 leading-relaxed">
							{preset.description}
						</p>
					</div>

					<div class="mt-3 pt-2 border-t border-slate-800/60 flex items-center justify-between text-[11px] text-slate-400 group-hover:text-indigo-300">
						<span>この文を読み込む</span>
						<span>→</span>
					</div>
				</button>
			{/each}
		</div>
	</div>
{/if}

