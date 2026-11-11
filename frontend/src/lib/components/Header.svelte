<script lang="ts">
	import type { Language } from '$lib/types';
	import { adminState } from '$lib/state/admin.svelte';
	import { Bookmark, Volume2, Sparkles, Globe2, ShieldCheck, Shield } from 'lucide-svelte';

	interface Props {
		languages: Language[];
		selectedLang: string;
		onSelectLang: (code: string) => void;
		onToggleBookmarks: () => void;
		bookmarkCount: number;
	}

	let { languages, selectedLang, onSelectLang, onToggleBookmarks, bookmarkCount }: Props = $props();

	let currentLangObj = $derived(
		languages.find(l => l.code === selectedLang) || { name: 'English (US)', flag: '🇺🇸' }
	);
</script>

<header class="sticky top-0 z-30 border-b border-slate-800 bg-slate-950/90 backdrop-blur-md px-4 lg:px-8 py-3.5 transition-all">
	<div class="max-w-7xl mx-auto flex items-center justify-between">
		<!-- Brand & Concept -->
		<div class="flex items-center gap-3">
			<div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-600 via-indigo-500 to-sky-400 flex items-center justify-center shadow-lg shadow-indigo-500/20 text-white font-bold text-xl tracking-wider">
				L
			</div>
			<div>
				<div class="flex items-center gap-2">
					<h1 class="text-xl font-bold bg-gradient-to-r from-white via-slate-100 to-indigo-200 bg-clip-text text-transparent">
						LinguaLight
					</h1>
					<span class="text-xs px-2 py-0.5 rounded-full font-medium bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 flex items-center gap-1">
						<Sparkles class="w-3 h-3" /> Svelte 5 + Go
					</span>
				</div>
				<p class="text-xs text-slate-400 hidden sm:block">多言語インタラクティブ学習 & リアルタイム音声発音ワークベンチ</p>
			</div>
		</div>

		<!-- Actions & Language Selector -->
		<div class="flex items-center gap-3">
			<!-- Language Dropdown -->
			<div class="relative">
				<label for="lang-select" class="sr-only">言語選択</label>
				<div class="flex items-center gap-2 bg-slate-900 border border-slate-800 hover:border-slate-700 rounded-lg px-3 py-1.5 text-sm transition-colors shadow-sm">
					<Globe2 class="w-4 h-4 text-slate-400" />
					<select
						id="lang-select"
						value={selectedLang}
						onchange={(e) => onSelectLang((e.target as HTMLSelectElement).value)}
						class="bg-transparent text-slate-200 font-medium focus:outline-none cursor-pointer pr-2"
					>
						{#each languages as lang}
							<option value={lang.code} class="bg-slate-900 text-white">
								{lang.flag} {lang.name}
							</option>
						{/each}
					</select>
				</div>
			</div>

			<!-- Admin Button -->
			<button
				onclick={() => (adminState.isModalOpen = true)}
				class="p-2 rounded-lg border transition-all flex items-center gap-1.5 text-xs font-semibold
					{adminState.isAdmin
						? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/20'
						: 'bg-slate-900 border-slate-800 text-slate-400 hover:text-white hover:border-slate-700'}"
				title={adminState.isAdmin ? '管理者コンソールを開く' : '管理者ログイン'}
			>
				{#if adminState.isAdmin}
					<ShieldCheck class="w-4 h-4 text-emerald-400" />
					<span class="hidden lg:inline">Admin</span>
				{:else}
					<Shield class="w-4 h-4" />
					<span class="hidden lg:inline">管理</span>
				{/if}
			</button>

			<!-- Bookmarks Toggle -->
			<button
				onclick={onToggleBookmarks}
				class="relative p-2 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 hover:text-white transition-all flex items-center gap-1.5 text-sm"
				title="保存した単語帳・フレーズ"
			>
				<Bookmark class="w-4 h-4 text-amber-400" />
				<span class="hidden md:inline font-medium">単語帳</span>
				{#if bookmarkCount > 0}
					<span class="ml-1 px-1.5 py-0.2 rounded-full text-xs font-semibold bg-indigo-600 text-white">
						{bookmarkCount}
					</span>
				{/if}
			</button>
		</div>
	</div>
</header>

