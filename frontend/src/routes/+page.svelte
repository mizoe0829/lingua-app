<script lang="ts">
	import { onMount } from 'svelte';
	import type { Language, PresetText, Sentence, Token, Bookmark } from '$lib/types';
	import Header from '$lib/components/Header.svelte';
	import AudioControls from '$lib/components/AudioControls.svelte';
	import TokenBoard from '$lib/components/TokenBoard.svelte';
	import AnalysisPanel from '$lib/components/AnalysisPanel.svelte';
	import PresetSelector from '$lib/components/PresetSelector.svelte';
	import BookmarksDrawer from '$lib/components/BookmarksDrawer.svelte';
	import AdminModal from '$lib/components/AdminModal.svelte';
	import SpeakingDialogModal from '$lib/components/SpeakingDialogModal.svelte';
	import { speechEngine } from '$lib/audio.svelte';
	import { adminState } from '$lib/state/admin.svelte';
	import { Sparkles, RefreshCw, Type, Mic, Play, CheckCircle2, MessageSquare } from 'lucide-svelte';

	// Svelte 5 Runes States
	let languages = $state<Language[]>([]);
	let selectedLang = $state('en-US');
	let presets = $state<PresetText[]>([]);
	let bookmarks = $state<Bookmark[]>([]);
	let isBookmarksOpen = $state(false);
	let isDialogOpen = $state(false);

	let rawInputText = $state(
		"Hi there! Could I get an oat milk flat white with an extra shot? Also, do you happen to have any fresh almond croissants left today?"
	);

	let sentences = $state<Sentence[]>([]);
	let wordCount = $state(0);
	let isTokenizing = $state(false);

	// Analysis state
	let activeAnalyzeSentence = $state<Sentence | null>(null);
	let activeAnalyzeToken = $state<Token | null>(null);
	let streamContent = $state('');
	let isStreaming = $state(false);
	let abortController: AbortController | null = null;

	// Toast state
	let toastMessage = $state<string | null>(null);

	function showToast(msg: string) {
		toastMessage = msg;
		setTimeout(() => {
			if (toastMessage === msg) toastMessage = null;
		}, 3000);
	}

	onMount(async () => {
		await loadLanguages();
		await loadPresets(selectedLang);
		await loadBookmarks();
		await handleTokenize();
	});

	async function loadLanguages() {
		try {
			const res = await fetch('/api/languages');
			if (res.ok) {
				languages = await res.json();
			}
		} catch (e) {
			console.error('Failed to load languages', e);
		}
	}

	async function loadPresets(lang: string) {
		try {
			const res = await fetch(`/api/presets?lang=${lang}`);
			if (res.ok) {
				presets = await res.json();
			}
		} catch (e) {
			console.error('Failed to load presets', e);
		}
	}

	async function loadBookmarks() {
		try {
			const res = await fetch('/api/bookmarks');
			if (res.ok) {
				bookmarks = await res.json();
			}
		} catch (e) {
			console.error('Failed to load bookmarks', e);
		}
	}

	async function handleSelectLang(code: string) {
		selectedLang = code;
		speechEngine.selectedLang = code;
		await loadPresets(code);
	}

	function handleSelectPreset(preset: PresetText) {
		rawInputText = preset.content;
		selectedLang = preset.lang_code;
		speechEngine.selectedLang = preset.lang_code;
		handleTokenize();
		showToast(`プリセット「${preset.title}」を読み込みました`);
	}

	async function handleTokenize() {
		if (!rawInputText.trim()) {
			sentences = [];
			wordCount = 0;
			return;
		}

		isTokenizing = true;
		try {
			const res = await fetch('/api/tokenize', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					lang_code: selectedLang,
					text: rawInputText
				})
			});

			if (res.ok) {
				const data = await res.json();
				sentences = data.sentences || [];
				wordCount = data.word_count || 0;
			}
		} catch (e) {
			console.error('Failed to tokenize', e);
		} finally {
			isTokenizing = false;
		}
	}

	async function handleAnalyzeSentence(sentence: Sentence, token?: Token) {
		activeAnalyzeSentence = sentence;
		activeAnalyzeToken = token || null;
		streamContent = '';
		isStreaming = true;

		if (abortController) {
			abortController.abort();
		}
		abortController = new AbortController();

		try {
			const res = await fetch('/api/analyze/stream', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					lang_code: selectedLang,
					sentence: sentence.raw_text,
					word: token ? token.clean_text : ''
				}),
				signal: abortController.signal
			});

			if (!res.ok) {
				streamContent = 'エラーが発生しました: ' + (await res.text());
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
								if (parsed.chunk) {
									streamContent += parsed.chunk;
								}
								if (parsed.done) {
									isStreaming = false;
								}
							} catch {
								// Ignore malformed chunks
							}
						}
					}
				}
			}
		} catch (err: any) {
			if (err.name !== 'AbortError') {
				streamContent += '\n[通信が切断されました]';
			}
		} finally {
			isStreaming = false;
		}
	}

	async function handleBookmarkWord(token: Token, sentenceText: string) {
		try {
			const res = await fetch('/api/bookmarks', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					lang_code: selectedLang,
					text: token.clean_text,
					ipa: token.ipa,
					meaning: token.meaning,
					context: sentenceText,
					notes: `${token.pos} (${selectedLang})`
				})
			});

			if (res.ok) {
				const newBookmark = await res.json();
				bookmarks = [newBookmark, ...bookmarks];
				showToast(`「${token.clean_text}」を単語帳に保存しました！`);
			}
		} catch (e) {
			console.error('Failed to bookmark', e);
		}
	}

	async function handleDeleteBookmark(id: number) {
		try {
			const res = await fetch(`/api/bookmarks/${id}`, { method: 'DELETE' });
			if (res.ok) {
				bookmarks = bookmarks.filter(b => b.id !== id);
				showToast('単語帳から削除しました');
			}
		} catch (e) {
			console.error('Failed to delete bookmark', e);
		}
	}

	function handlePresetCreated(newPreset: PresetText) {
		if (newPreset.lang_code === selectedLang) {
			presets = [newPreset, ...presets];
		}
		showToast(`例文「${newPreset.title}」を追加しました！`);
	}

	async function handleDeletePreset(id: string) {
		try {
			const res = await fetch(`/api/presets/${id}`, {
				method: 'DELETE',
				headers: {
					'X-Admin-Secret': adminState.adminSecret
				}
			});

			if (res.ok) {
				presets = presets.filter(p => p.id !== id);
				showToast('例文を削除しました');
			} else {
				showToast('削除に失敗しました: ' + (await res.text()));
			}
		} catch (e) {
			console.error('Failed to delete preset', e);
			showToast('通信エラーが発生しました');
		}
	}
</script>

<!-- Header -->
<Header
	{languages}
	{selectedLang}
	onSelectLang={handleSelectLang}
	onToggleBookmarks={() => (isBookmarksOpen = !isBookmarksOpen)}
	onOpenDialog={() => (isDialogOpen = true)}
	bookmarkCount={bookmarks.length}
/>

<!-- Main Content -->
<main class="flex-1 max-w-7xl w-full mx-auto px-4 lg:px-8 py-8 space-y-8">
	<!-- Hero Intro -->
	<section class="text-center max-w-3xl mx-auto space-y-4">
		<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
			<Sparkles class="w-3.5 h-3.5" /> Zero Latency Speech & Phonetics Workbench
		</div>
		<h2 class="text-3xl sm:text-4xl font-extrabold tracking-tight text-white">
			話す・聞く力を 0 秒で鍛える、<br />
			<span class="bg-gradient-to-r from-indigo-400 via-sky-300 to-indigo-200 bg-clip-text text-transparent">
				多言語インタラクティブ・ワークベンチ
			</span>
		</h2>
		<p class="text-slate-400 text-sm sm:text-base leading-relaxed">
			単語をタップした瞬間に 0ms でネイティブ発音。Svelte 5 Runes が実現する仮想DOMレスの即時ハイライトと、Go + Gemini AI による音の繋がり（リエゾン）・イントネーション・構文ストリーミング解説。
		</p>

		<!-- Hero CTA -->
		<div class="flex items-center justify-center gap-3 pt-2">
			<button
				onclick={() => (isDialogOpen = true)}
				class="flex items-center gap-2 px-5 py-2.5 rounded-2xl bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-bold text-sm shadow-xl shadow-sky-500/20 transition-all hover:scale-[1.03] active:scale-95"
			>
				<MessageSquare class="w-4 h-4" />
				<span>AI壁打ち対話をはじめる</span>
				<span class="text-xs px-2 py-0.5 rounded-full bg-white/20 font-normal">習熟度レベル別</span>
			</button>
		</div>
	</section>

	<!-- Presets Quick Bar -->
	<PresetSelector {presets} onSelectPreset={handleSelectPreset} onDeletePreset={handleDeletePreset} />

	<!-- Text Input & Tokenize Bar -->
	<section class="space-y-3 bg-slate-900/60 p-5 rounded-2xl border border-slate-800 shadow-xl">
		<div class="flex items-center justify-between">
			<label for="learning-text" class="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
				<Type class="w-4 h-4 text-indigo-400" />
				学習テキスト入力（任意の文章をペースト可能）
			</label>
			<div class="text-xs text-slate-400 flex items-center gap-2">
				<span>単語数: <strong class="text-indigo-300">{wordCount}</strong></span>
				<span>•</span>
				<span>文数: <strong class="text-indigo-300">{sentences.length}</strong></span>
			</div>
		</div>

		<textarea
			id="learning-text"
			bind:value={rawInputText}
			rows="3"
			placeholder="学習したい英文や他言語のテキストを入力してください..."
			class="w-full bg-slate-950 border border-slate-800 rounded-xl p-3.5 text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 text-sm sm:text-base transition-all resize-y leading-relaxed"
		></textarea>

		<div class="flex items-center justify-between pt-1">
			<p class="text-xs text-slate-500 hidden sm:block">
				※ 入力内容をトークン分解し、IPA国際音声記号・品詞・意味を即時マッピングします。
			</p>
			<button
				onclick={handleTokenize}
				disabled={isTokenizing}
				class="flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-sm shadow-lg shadow-indigo-600/30 transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-50 ml-auto"
			>
				<RefreshCw class="w-4 h-4 {isTokenizing ? 'animate-spin' : ''}" />
				<span>トークン解析を実行</span>
			</button>
		</div>
	</section>

	<!-- Audio Controller Bar -->
	<AudioControls onStopAll={() => {}} />

	<!-- Main Workbench Grid: Token Board & AI Analysis Panel -->
	<section class="grid grid-cols-1 {activeAnalyzeSentence ? 'lg:grid-cols-12' : 'lg:grid-cols-1'} gap-6 items-start">
		<!-- Left: Interactive Tokens Board -->
		<div class="{activeAnalyzeSentence ? 'lg:col-span-7' : 'lg:col-span-12'} space-y-4">
			<div class="flex items-center justify-between pb-2 border-b border-slate-800">
				<h3 class="font-bold text-slate-200 flex items-center gap-2 text-sm sm:text-base">
					<Mic class="w-4 h-4 text-emerald-400" />
					インタラクティブ発音ボード
				</h3>
				<span class="text-xs text-slate-400">
					単語タップで発音 / ホバーで辞書表示
				</span>
			</div>

			<TokenBoard
				{sentences}
				langCode={selectedLang}
				onAnalyzeSentence={handleAnalyzeSentence}
				onBookmarkWord={handleBookmarkWord}
			/>
		</div>

		<!-- Right: AI Analysis Panel -->
		{#if activeAnalyzeSentence}
			<div class="lg:col-span-5 sticky top-20">
				<AnalysisPanel
					sentence={activeAnalyzeSentence}
					focusedToken={activeAnalyzeToken}
					langCode={selectedLang}
					{streamContent}
					{isStreaming}
					onClose={() => (activeAnalyzeSentence = null)}
				/>
			</div>
		{/if}
	</section>
</main>

<!-- Bookmarks Drawer -->
<BookmarksDrawer
	isOpen={isBookmarksOpen}
	{bookmarks}
	{selectedLang}
	onClose={() => (isBookmarksOpen = false)}
	onDelete={handleDeleteBookmark}
/>

<!-- Admin Modal -->
<AdminModal
	{languages}
	{selectedLang}
	onPresetCreated={handlePresetCreated}
/>

<!-- Speaking Dialog Sparring Modal -->
<SpeakingDialogModal
	isOpen={isDialogOpen}
	{selectedLang}
	{languages}
	onClose={() => (isDialogOpen = false)}
/>

<!-- Toast Notification -->
{#if toastMessage}
	<div
		class="fixed bottom-6 right-6 z-50 flex items-center gap-2 px-4 py-2.5 rounded-xl bg-slate-900 border border-indigo-500/40 text-slate-100 shadow-2xl shadow-indigo-500/20 text-sm animate-in fade-in slide-in-from-bottom-3"
	>
		<CheckCircle2 class="w-4 h-4 text-emerald-400" />
		<span>{toastMessage}</span>
	</div>
{/if}

