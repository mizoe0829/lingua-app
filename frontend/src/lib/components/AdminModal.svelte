<script lang="ts">
	import { adminState } from '$lib/state/admin.svelte';
	import type { Language, PresetText } from '$lib/types';
	import { X, ShieldCheck, Lock, Plus, CheckCircle2, AlertCircle } from 'lucide-svelte';

	interface Props {
		languages: Language[];
		selectedLang: string;
		onPresetCreated: (preset: PresetText) => void;
	}

	let { languages, selectedLang, onPresetCreated }: Props = $props();

	// Login form state
	let inputSecret = $state('');
	let loginError = $state('');
	let isSubmitting = $state(false);

	// Create preset form state
	let newLangCode = $state('en-US');
	$effect(() => {
		newLangCode = selectedLang;
	});
	let newTitle = $state('');
	let newCategory = $state('Daily Conversation');
	let newDifficulty = $state<'Beginner' | 'Intermediate' | 'Advanced'>('Intermediate');
	let newDescription = $state('');
	let newContent = $state('');
	let createError = $state('');
	let createSuccess = $state('');

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		loginError = '';
		isSubmitting = true;

		const success = await adminState.verify(inputSecret.trim());
		isSubmitting = false;

		if (!success) {
			loginError = '管理者シークレットキーが正しくありません。';
		} else {
			inputSecret = '';
		}
	}

	async function handleCreatePreset(e: SubmitEvent) {
		e.preventDefault();
		createError = '';
		createSuccess = '';

		if (!newTitle.trim() || !newContent.trim()) {
			createError = 'タイトルと本文は必須です。';
			return;
		}

		isSubmitting = true;
		try {
			const res = await fetch('/api/presets', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'X-Admin-Secret': adminState.adminSecret
				},
				body: JSON.stringify({
					lang_code: newLangCode,
					category: newCategory,
					title: newTitle.trim(),
					description: newDescription.trim() || 'Custom administrative learning preset.',
					content: newContent.trim(),
					difficulty: newDifficulty
				})
			});

			if (res.ok) {
				const created = await res.json();
				onPresetCreated(created);
				createSuccess = `例文「${created.title}」を正常に追加しました！`;
				// Reset form
				newTitle = '';
				newContent = '';
				newDescription = '';
				setTimeout(() => {
					adminState.isModalOpen = false;
					createSuccess = '';
				}, 1200);
			} else {
				createError = '追加に失敗しました: ' + (await res.text());
			}
		} catch (err: any) {
			createError = '通信エラーが発生しました: ' + err.message;
		} finally {
			isSubmitting = false;
		}
	}
</script>

{#if adminState.isModalOpen}
	<!-- Backdrop -->
	<div
		tabindex="0"
		role="button"
		class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in"
		onclick={() => (adminState.isModalOpen = false)}
		onkeydown={(e) => e.key === 'Escape' && (adminState.isModalOpen = false)}
	>
		<!-- Modal Content -->
		<div
			tabindex="-1"
			role="dialog"
			aria-modal="true"
			class="relative w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl p-6 overflow-hidden space-y-5"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<!-- Header -->
			<div class="flex items-center justify-between pb-3 border-b border-slate-800">
				<div class="flex items-center gap-2.5">
					<div class="p-2 rounded-xl bg-indigo-600/20 text-indigo-400 border border-indigo-500/30">
						<ShieldCheck class="w-5 h-5" />
					</div>
					<div>
						<h3 class="font-bold text-slate-100 text-lg">
							{adminState.isAdmin ? '管理者コンソール — 例文の追加' : '管理者ログイン'}
						</h3>
						<p class="text-xs text-slate-400">
							{adminState.isAdmin ? '新しい学習プリセット文を SQLite に永続化' : '例文の追加・削除には管理者権限が必要です'}
						</p>
					</div>
				</div>

				<button
					onclick={() => (adminState.isModalOpen = false)}
					class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Form Area -->
			{#if !adminState.isAdmin}
				<!-- Login Form -->
				<form onsubmit={handleLogin} class="space-y-4">
					<div>
						<label for="admin-secret" class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1.5">
							管理者シークレットキー
						</label>
						<div class="relative">
							<input
								id="admin-secret"
								type="password"
								bind:value={inputSecret}
								placeholder="デフォルト: lingua-admin-2026"
								class="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
							/>
							<Lock class="w-4 h-4 text-slate-500 absolute right-3.5 top-3" />
						</div>
						<p class="text-[11px] text-slate-500 mt-1">※ デフォルト環境設定キー: <code class="text-indigo-400">lingua-admin-2026</code></p>
					</div>

					{#if loginError}
						<div class="flex items-center gap-2 p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs">
							<AlertCircle class="w-4 h-4 shrink-0" />
							<span>{loginError}</span>
						</div>
					{/if}

					<div class="flex items-center justify-end gap-2 pt-2">
						<button
							type="button"
							onclick={() => (adminState.isModalOpen = false)}
							class="px-4 py-2 rounded-xl text-xs font-semibold text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
						>
							キャンセル
						</button>
						<button
							type="submit"
							disabled={isSubmitting || !inputSecret.trim()}
							class="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/30 transition-all disabled:opacity-50"
						>
							{isSubmitting ? '認証中...' : 'ログイン'}
						</button>
					</div>
				</form>
			{:else}
				<!-- Create Preset Form -->
				<form onsubmit={handleCreatePreset} class="space-y-4">
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="preset-lang" class="block text-xs font-semibold text-slate-300 mb-1">言語</label>
							<select
								id="preset-lang"
								bind:value={newLangCode}
								class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
							>
								{#each languages as l}
									<option value={l.code}>{l.flag} {l.name}</option>
								{/each}
							</select>
						</div>

						<div>
							<label for="preset-difficulty" class="block text-xs font-semibold text-slate-300 mb-1">難易度</label>
							<select
								id="preset-difficulty"
								bind:value={newDifficulty}
								class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500"
							>
								<option value="Beginner">Beginner (初級)</option>
								<option value="Intermediate">Intermediate (中級)</option>
								<option value="Advanced">Advanced (上級)</option>
							</select>
						</div>
					</div>

					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="preset-category" class="block text-xs font-semibold text-slate-300 mb-1">カテゴリ</label>
							<input
								id="preset-category"
								type="text"
								bind:value={newCategory}
								placeholder="Daily, Business, Tech..."
								class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
							/>
						</div>

						<div>
							<label for="preset-title" class="block text-xs font-semibold text-slate-300 mb-1">タイトル *</label>
							<input
								id="preset-title"
								type="text"
								bind:value={newTitle}
								placeholder="例: Tech Conference Q&A"
								class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
							/>
						</div>
					</div>

					<div>
						<label for="preset-desc" class="block text-xs font-semibold text-slate-300 mb-1">学習の狙い・説明</label>
						<input
							id="preset-desc"
							type="text"
							bind:value={newDescription}
							placeholder="例: 会話のリズムと音の脱落に注目した例文"
							class="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
						/>
					</div>

					<div>
						<label for="preset-content" class="block text-xs font-semibold text-slate-300 mb-1">例文本文 (外国語) *</label>
						<textarea
							id="preset-content"
							bind:value={newContent}
							rows="3"
							placeholder="学習対象のテキストを入力してください..."
							class="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-xs sm:text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 leading-relaxed"
						></textarea>
					</div>

					{#if createError}
						<div class="flex items-center gap-2 p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs">
							<AlertCircle class="w-4 h-4 shrink-0" />
							<span>{createError}</span>
						</div>
					{/if}

					{#if createSuccess}
						<div class="flex items-center gap-2 p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-xs">
							<CheckCircle2 class="w-4 h-4 shrink-0" />
							<span>{createSuccess}</span>
						</div>
					{/if}

					<div class="flex items-center justify-between pt-2 border-t border-slate-800">
						<button
							type="button"
							onclick={() => adminState.logout()}
							class="text-xs text-slate-500 hover:text-rose-400 transition-colors"
						>
							ログアウト
						</button>

						<div class="flex items-center gap-2">
							<button
								type="button"
								onclick={() => (adminState.isModalOpen = false)}
								class="px-4 py-2 rounded-xl text-xs font-semibold text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
							>
								閉じる
							</button>
							<button
								type="submit"
								disabled={isSubmitting || !newTitle.trim() || !newContent.trim()}
								class="flex items-center gap-1.5 px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/30 transition-all disabled:opacity-50"
							>
								<Plus class="w-4 h-4" />
								<span>{isSubmitting ? '保存中...' : '例文を追加'}</span>
							</button>
						</div>
					</div>
				</form>
			{/if}
		</div>
	</div>
{/if}

