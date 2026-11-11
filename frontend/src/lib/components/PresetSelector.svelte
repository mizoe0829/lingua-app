<script lang="ts">
	import type { PresetText } from '$lib/types';
	import { adminState } from '$lib/state/admin.svelte';
	import { BookOpen, Plus, Trash2, Shield } from 'lucide-svelte';

	interface Props {
		presets: PresetText[];
		onSelectPreset: (preset: PresetText) => void;
		onDeletePreset?: (id: string) => void;
	}

	let { presets, onSelectPreset, onDeletePreset }: Props = $props();

	function handleDelete(e: MouseEvent, id: string) {
		e.stopPropagation();
		if (!confirm('この例文を削除してもよろしいですか？')) return;
		if (onDeletePreset) onDeletePreset(id);
	}
</script>

<div class="space-y-3">
	<div class="flex items-center justify-between">
		<h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
			<BookOpen class="w-3.5 h-3.5 text-indigo-400" />
			学習用プリセット文 (ワンタップで学習開始)
		</h3>

		<div class="flex items-center gap-2">
			{#if adminState.isAdmin}
				<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
					<Shield class="w-3 h-3" /> 管理者モード
				</span>
				<button
					type="button"
					onclick={() => (adminState.isModalOpen = true)}
					class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/30 text-xs font-semibold transition-all hover:scale-[1.02]"
				>
					<Plus class="w-3.5 h-3.5" /> 例文を追加
				</button>
			{:else}
				<button
					type="button"
					onclick={() => (adminState.isModalOpen = true)}
					class="px-2 py-0.5 rounded-lg border border-slate-800 bg-slate-900/80 text-xs font-medium text-slate-400 hover:text-white hover:border-slate-700 transition-colors flex items-center gap-1"
				>
					<Shield class="w-3.5 h-3.5 text-slate-400" /> 管理者ログイン
				</button>
			{/if}
		</div>
	</div>

	{#if presets.length > 0}
		<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
			{#each presets as preset (preset.id)}
				<div
					tabindex="0"
					role="button"
					onclick={() => onSelectPreset(preset)}
					onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && onSelectPreset(preset)}
					class="text-left p-3.5 rounded-xl bg-slate-900/60 border border-slate-800 hover:border-indigo-500/50 hover:bg-slate-900 transition-all duration-200 group flex flex-col justify-between relative cursor-pointer"
				>
					<div>
						<div class="flex items-center justify-between mb-1.5">
							<span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700">
								{preset.category}
							</span>
							<div class="flex items-center gap-1.5">
								<span class="text-[10px] font-semibold text-indigo-400">
									{preset.difficulty}
								</span>
								{#if adminState.isAdmin}
									<button
										type="button"
										onclick={(e) => handleDelete(e, preset.id)}
										class="p-1 rounded hover:bg-rose-500/20 text-slate-500 hover:text-rose-400 transition-colors"
										title="この例文を削除"
									>
										<Trash2 class="w-3.5 h-3.5" />
									</button>
								{/if}
							</div>
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
				</div>
			{/each}
		</div>
	{:else}
		<div class="p-6 text-center text-xs text-slate-500 bg-slate-900/40 rounded-xl border border-slate-800">
			この言語のプリセット文はまだありません。
			{#if adminState.isAdmin}
				上の「例文を追加」ボタンから新しく作成できます。
			{/if}
		</div>
	{/if}
</div>
