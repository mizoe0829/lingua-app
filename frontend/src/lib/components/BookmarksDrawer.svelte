<script lang="ts">
	import type { Bookmark } from '$lib/types';
	import { X, Volume2, Trash2, Bookmark as BookmarkIcon } from 'lucide-svelte';
	import { speechEngine } from '$lib/audio.svelte';

	interface Props {
		isOpen: boolean;
		bookmarks: Bookmark[];
		selectedLang: string;
		onClose: () => void;
		onDelete: (id: number) => void;
	}

	let { isOpen, bookmarks, selectedLang, onClose, onDelete }: Props = $props();

	function handleSpeak(text: string, langCode: string) {
		speechEngine.speakWord(text, langCode);
	}
</script>

{#if isOpen}
	<!-- Backdrop -->
	<div
		tabindex="0"
		role="button"
		class="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm transition-opacity"
		onclick={onClose}
		onkeydown={(e) => e.key === 'Escape' && onClose()}
	></div>

	<!-- Drawer panel -->
	<aside
		class="fixed top-0 right-0 z-50 h-full w-full sm:w-96 bg-slate-900 border-l border-slate-800 shadow-2xl p-5 flex flex-col justify-between transition-transform transform translate-x-0"
	>
		<div>
			<div class="flex items-center justify-between pb-4 border-b border-slate-800">
				<div class="flex items-center gap-2">
					<div class="p-2 rounded-lg bg-amber-400/10 text-amber-400">
						<BookmarkIcon class="w-5 h-5" />
					</div>
					<div>
						<h3 class="font-bold text-slate-100">マイスペース (単語帳)</h3>
						<p class="text-xs text-slate-400">保存済みアイテム: {bookmarks.length} 件</p>
					</div>
				</div>

				<button
					onclick={onClose}
					class="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
					title="閉じる"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Bookmark List -->
			<div class="mt-4 space-y-3 overflow-y-auto max-h-[calc(100vh-140px)] pr-1">
				{#if bookmarks.length === 0}
					<div class="text-center py-12 text-slate-500 text-sm">
						保存された単語やフレーズはまだありません。<br />
						単語にホバーして「＋」ボタンを押すと保存できます。
					</div>
				{:else}
					{#each bookmarks as b (b.id)}
						<div class="p-3.5 rounded-xl bg-slate-950/60 border border-slate-800/80 hover:border-slate-700 space-y-2 group">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2">
									<span class="font-bold text-white text-base">{b.text}</span>
									{#if b.ipa}
										<span class="font-mono text-xs text-indigo-300 bg-indigo-950 px-1 rounded">{b.ipa}</span>
									{/if}
								</div>

								<div class="flex items-center gap-1">
									<button
										onclick={() => handleSpeak(b.text, b.lang_code)}
										class="p-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 transition-colors"
										title="発音を再生"
									>
										<Volume2 class="w-4 h-4" />
									</button>
									{#if b.id}
										<button
											onclick={() => onDelete(b.id!)}
											class="p-1.5 rounded-lg hover:bg-rose-500/20 text-slate-500 hover:text-rose-400 transition-colors"
											title="削除"
										>
											<Trash2 class="w-4 h-4" />
										</button>
									{/if}
								</div>
							</div>

							{#if b.meaning}
								<p class="text-xs text-slate-300">{b.meaning}</p>
							{/if}

							{#if b.context}
								<p class="text-[11px] text-slate-500 italic line-clamp-1 border-t border-slate-800/60 pt-1">
									"{b.context}"
								</p>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		</div>

		<div class="pt-4 border-t border-slate-800 text-center text-xs text-slate-400">
			SQLite WALモードでローカル高速永続化
		</div>
	</aside>
{/if}

