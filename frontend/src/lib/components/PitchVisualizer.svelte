<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Sentence } from '$lib/types';
	import { Mic, Square, Play, Sparkles, Activity, CheckCircle, RotateCcw, Volume2 } from 'lucide-svelte';
	import { speechEngine } from '$lib/audio.svelte';

	interface Props {
		sentence: Sentence;
		langCode: string;
		onClose: () => void;
	}

	let { sentence, langCode, onClose }: Props = $props();

	let canvasRef = $state<HTMLCanvasElement | null>(null);
	let isRecording = $state(false);
	let matchScore = $state<number | null>(null);
	let feedbackText = $state<string>('');
	let userPitchPoints = $state<{ t: number; hz: number }[]>([]);
	let targetPitchPoints = $state<{ t: number; hz: number; label?: string }[]>([]);

	let audioCtx: AudioContext | null = null;
	let analyser: AnalyserNode | null = null;
	let micStream: MediaStream | null = null;
	let animId: number | null = null;
	let recordStartTime = 0;
	const durationSeconds = 4.0; // Fixed visual timeline span

	onMount(() => {
		generateTargetContour();
		drawContour();
	});

	onDestroy(() => {
		stopRecording();
		if (audioCtx) {
			audioCtx.close();
			audioCtx = null;
		}
	});

	/**
	 * Generate native baseline intonation contour based on tokens and punctuation
	 */
	function generateTargetContour() {
		const points: { t: number; hz: number; label?: string }[] = [];
		const isQuestion = sentence.raw_text.trim().endsWith('?');
		const words = sentence.tokens.filter(t => t.is_word);
		const totalWords = Math.max(words.length, 1);

		// Time slice per word
		const margin = 0.4;
		const availableTime = durationSeconds - margin * 2;
		const step = availableTime / totalWords;

		// Base frequency (around 160Hz for neutral, 220Hz peak, 120Hz floor)
		const baseHz = 160;

		for (let i = 0; i < totalWords; i++) {
			const w = words[i];
			const tStart = margin + i * step;
			const tMid = tStart + step * 0.5;
			const isImportant = w.ipa && (w.ipa.includes('ˈ') || w.pos === 'NOUN' || w.pos === 'VERB');

			// Stressed words peak higher
			let peakHz = baseHz;
			if (isImportant) {
				peakHz = baseHz + 50 + (i % 2 === 0 ? 15 : -10);
			} else {
				peakHz = baseHz - 20;
			}

			// Question sentence-final rise
			if (i === totalWords - 1 && isQuestion) {
				peakHz = baseHz + 75;
			} else if (i === totalWords - 1 && !isQuestion) {
				// Statement-final fall
				peakHz = baseHz - 35;
			}

			points.push({ t: tStart, hz: baseHz, label: w.text });
			points.push({ t: tMid, hz: peakHz });
			points.push({ t: tStart + step * 0.9, hz: baseHz - 10 });
		}

		targetPitchPoints = points;
	}

	async function startRecording() {
		try {
			if (!audioCtx) {
				audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
			}
			if (audioCtx.state === 'suspended') {
				await audioCtx.resume();
			}

			micStream = await navigator.mediaDevices.getUserMedia({ audio: true });
			const source = audioCtx.createMediaStreamSource(micStream);
			analyser = audioCtx.createAnalyser();
			analyser.fftSize = 2048;
			source.connect(analyser);

			userPitchPoints = [];
			matchScore = null;
			feedbackText = '';
			isRecording = true;
			recordStartTime = performance.now();

			// Auto play model pronunciation as reference if desired
			speechEngine.speakSentence(sentence.id, sentence.raw_text, langCode, sentence.tokens);

			trackPitchLoop();
		} catch (err: any) {
			console.error('Mic access error:', err);
			feedbackText = 'マイクへのアクセスが許可されていません';
			isRecording = false;
		}
	}

	function stopRecording() {
		if (!isRecording) return;
		isRecording = false;

		if (animId) {
			cancelAnimationFrame(animId);
			animId = null;
		}
		if (micStream) {
			micStream.getTracks().forEach(t => t.stop());
			micStream = null;
		}

		calculateScore();
		drawContour();
	}

	function trackPitchLoop() {
		if (!isRecording || !analyser || !audioCtx) return;

		const buffer = new Float32Array(analyser.fftSize);
		analyser.getFloatTimeDomainData(buffer);

		const elapsed = (performance.now() - recordStartTime) / 1000;
		if (elapsed >= durationSeconds) {
			stopRecording();
			return;
		}

		const pitch = autoCorrelate(buffer, audioCtx.sampleRate);
		if (pitch > 70 && pitch < 380) {
			userPitchPoints.push({ t: elapsed, hz: pitch });
		}

		drawContour(elapsed);
		animId = requestAnimationFrame(trackPitchLoop);
	}

	/**
	 * Autocorrelation algorithm for pitch detection
	 */
	function autoCorrelate(buf: Float32Array, sampleRate: number): number {
		let size = buf.length;
		let rms = 0;
		for (let i = 0; i < size; i++) {
			rms += buf[i] * buf[i];
		}
		rms = Math.sqrt(rms / size);
		if (rms < 0.015) return -1; // Volume too quiet

		let r1 = 0, r2 = size - 1, thres = 0.2;
		for (let i = 0; i < size / 2; i++) {
			if (Math.abs(buf[i]) < thres) { r1 = i; break; }
		}
		for (let i = 1; i < size / 2; i++) {
			if (Math.abs(buf[size - i]) < thres) { r2 = size - i; break; }
		}
		buf = buf.slice(r1, r2);
		size = buf.length;

		const c = new Float32Array(size);
		for (let i = 0; i < size; i++) {
			for (let j = 0; j < size - i; j++) {
				c[i] = c[i] + buf[j] * buf[j + i];
			}
		}

		let d = 0;
		while (c[d] > c[d + 1]) d++;
		let maxval = -1, maxpos = -1;
		for (let i = d; i < size; i++) {
			if (c[i] > maxval) {
				maxval = c[i];
				maxpos = i;
			}
		}

		const T0 = maxpos;
		return sampleRate / T0;
	}

	function calculateScore() {
		if (userPitchPoints.length < 5) {
			matchScore = 65;
			feedbackText = '音声が小さかったか短時間でした。もう少し大きめの声ではっきりと発声してみましょう！';
			return;
		}

		// Calculate similarity curve trend
		let diffSum = 0;
		let validCount = 0;

		for (const u of userPitchPoints) {
			// Find closest target point in time
			const nearest = targetPitchPoints.reduce((prev, curr) =>
				Math.abs(curr.t - u.t) < Math.abs(prev.t - u.t) ? curr : prev
			);
			if (nearest) {
				const hzDiff = Math.abs(u.hz - nearest.hz);
				diffSum += Math.min(hzDiff, 100);
				validCount++;
			}
		}

		const avgDiff = validCount > 0 ? diffSum / validCount : 50;
		const rawScore = Math.max(50, Math.min(98, Math.round(100 - avgDiff * 0.65)));
		matchScore = rawScore;

		const isQuestion = sentence.raw_text.trim().endsWith('?');
		if (rawScore >= 85) {
			feedbackText = isQuestion
				? '🎉 素晴らしい！疑問文特有の語尾の上昇（Rising Intonation）と強勢の山が綺麗に決まっています！'
				: '🎉 Excellent! 文全体のイントネーションの高低差と、文末の下げ（Falling Intonation）が自然です！';
		} else if (rawScore >= 70) {
			feedbackText = '👍 良好なピッチリズムです。主要な名詞や動詞で少しピッチ（声の高さ）を上げるとさらに自然になります。';
		} else {
			feedbackText = '💡 平坦（モノトーン）にならず、リズムの山と谷を意識して手本を聴きながら真似してみましょう。';
		}
	}

	function drawContour(currentElapsed: number = 0) {
		const canvas = canvasRef;
		if (!canvas) return;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;

		const w = canvas.width;
		const h = canvas.height;
		const minHz = 80;
		const maxHz = 320;

		// Clear canvas with dark gradient
		ctx.fillStyle = '#090d16';
		ctx.fillRect(0, 0, w, h);

		// Grid lines (100Hz, 150Hz, 200Hz, 250Hz, 300Hz)
		ctx.strokeStyle = '#1e293b';
		ctx.lineWidth = 1;
		ctx.font = '10px monospace';
		ctx.fillStyle = '#475569';

		for (let hz = 100; hz <= 300; hz += 50) {
			const y = h - ((hz - minHz) / (maxHz - minHz)) * h;
			ctx.beginPath();
			ctx.moveTo(40, y);
			ctx.lineTo(w, y);
			ctx.stroke();
			ctx.fillText(`${hz}Hz`, 6, y + 3);
		}

		const timeToX = (t: number) => 45 + (t / durationSeconds) * (w - 55);
		const hzToY = (hz: number) => h - ((hz - minHz) / (maxHz - minHz)) * h;

		// 1. Draw Target Model Intonation (Cyan dashed line)
		if (targetPitchPoints.length > 1) {
			ctx.strokeStyle = '#38bdf8';
			ctx.lineWidth = 3;
			ctx.setLineDash([6, 4]);
			ctx.beginPath();
			ctx.moveTo(timeToX(targetPitchPoints[0].t), hzToY(targetPitchPoints[0].hz));

			for (let i = 1; i < targetPitchPoints.length; i++) {
				const pt = targetPitchPoints[i];
				ctx.lineTo(timeToX(pt.t), hzToY(pt.hz));
			}
			ctx.stroke();
			ctx.setLineDash([]);

			// Word labels on target curve
			ctx.fillStyle = '#bae6fd';
			ctx.font = '11px sans-serif';
			for (const pt of targetPitchPoints) {
				if (pt.label) {
					ctx.fillText(pt.label, timeToX(pt.t) - 10, hzToY(pt.hz) - 10);
					ctx.beginPath();
					ctx.arc(timeToX(pt.t), hzToY(pt.hz), 3, 0, Math.PI * 2);
					ctx.fillStyle = '#38bdf8';
					ctx.fill();
				}
			}
		}

		// 2. Draw User Microphone Pitch Points (Amber glow curve)
		if (userPitchPoints.length > 1) {
			ctx.strokeStyle = '#f59e0b';
			ctx.lineWidth = 3.5;
			ctx.shadowColor = '#f59e0b';
			ctx.shadowBlur = 8;
			ctx.beginPath();
			ctx.moveTo(timeToX(userPitchPoints[0].t), hzToY(userPitchPoints[0].hz));

			for (let i = 1; i < userPitchPoints.length; i++) {
				const pt = userPitchPoints[i];
				ctx.lineTo(timeToX(pt.t), hzToY(pt.hz));
			}
			ctx.stroke();
			ctx.shadowBlur = 0;
		}

		// 3. Playhead cursor during recording
		if (isRecording && currentElapsed > 0) {
			const cursorX = timeToX(currentElapsed);
			ctx.strokeStyle = '#ef4444';
			ctx.lineWidth = 2;
			ctx.beginPath();
			ctx.moveTo(cursorX, 0);
			ctx.lineTo(cursorX, h);
			ctx.stroke();
		}
	}
</script>

<div class="mt-4 p-5 rounded-2xl bg-slate-950 border border-indigo-500/30 shadow-2xl space-y-4 animate-in fade-in">
	<!-- Header -->
	<div class="flex items-center justify-between pb-3 border-b border-slate-800">
		<div class="flex items-center gap-2.5">
			<div class="p-2 rounded-xl bg-indigo-500/20 text-indigo-400">
				<Activity class="w-5 h-5" />
			</div>
			<div>
				<h4 class="font-bold text-white text-sm sm:text-base flex items-center gap-2">
					イントネーション・ピッチ曲線ビジュアライザー
					<span class="text-xs px-2 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 font-normal">Web Audio API</span>
				</h4>
				<p class="text-xs text-slate-400">
					青破線（ネイティブお手本）に合わせて発声し、オレンジ実線（あなたの声）の抑揚カーブを一致させましょう。
				</p>
			</div>
		</div>

		<button
			type="button"
			onclick={onClose}
			class="text-xs px-3 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-800 text-slate-400 hover:text-white border border-slate-800 transition-colors"
		>
			閉じる
		</button>
	</div>

	<!-- Canvas Area -->
	<div class="relative w-full rounded-xl overflow-hidden border border-slate-800 bg-slate-950">
		<canvas
			bind:this={canvasRef}
			width={800}
			height={240}
			class="w-full h-48 sm:h-56 block"
		></canvas>

		<!-- Legend Overlay -->
		<div class="absolute top-3 right-3 flex items-center gap-4 px-3 py-1.5 rounded-lg bg-slate-900/80 backdrop-blur-sm border border-slate-800 text-xs">
			<div class="flex items-center gap-1.5">
				<span class="w-3 h-1 rounded-full bg-sky-400 border border-sky-300"></span>
				<span class="text-sky-300">お手本ピッチ</span>
			</div>
			<div class="flex items-center gap-1.5">
				<span class="w-3 h-1 rounded-full bg-amber-400"></span>
				<span class="text-amber-300">あなたのピッチ</span>
			</div>
		</div>
	</div>

	<!-- Controls & Score -->
	<div class="flex flex-col sm:flex-row items-center justify-between gap-4 pt-1">
		<div class="flex items-center gap-3">
			{#if !isRecording}
				<button
					type="button"
					onclick={startRecording}
					class="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-500 text-white font-semibold text-sm shadow-lg shadow-rose-600/30 transition-all hover:scale-105 active:scale-95"
				>
					<Mic class="w-4 h-4" />
					<span>マイクで発声して比較</span>
				</button>
			{:else}
				<button
					type="button"
					onclick={stopRecording}
					class="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white font-semibold text-sm animate-pulse shadow-lg shadow-amber-600/30 transition-all"
				>
					<Square class="w-4 h-4 fill-white" />
					<span>録音を停止して採点</span>
				</button>
			{/if}

			<button
				type="button"
				onclick={() => speechEngine.speakSentence(sentence.id, sentence.raw_text, langCode, sentence.tokens)}
				class="flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white border border-slate-700 text-xs font-medium transition-colors"
			>
				<Volume2 class="w-4 h-4" />
				<span>お手本を聴く</span>
			</button>
		</div>

		{#if matchScore !== null}
			<div class="flex items-center gap-3 bg-indigo-500/10 px-4 py-2 rounded-xl border border-indigo-500/20">
				<div class="text-right">
					<span class="text-[11px] text-slate-400 block">ピッチ一致スコア</span>
					<span class="text-lg font-mono font-extrabold {matchScore >= 80 ? 'text-emerald-400' : matchScore >= 70 ? 'text-amber-400' : 'text-rose-400'}">
						{matchScore}<span class="text-xs">/100</span>
					</span>
				</div>
				<CheckCircle class="w-5 h-5 text-indigo-400" />
			</div>
		{/if}
	</div>

	<!-- Feedback message -->
	{#if feedbackText}
		<div class="p-3.5 rounded-xl bg-indigo-950/40 border border-indigo-500/20 text-xs text-slate-300 leading-relaxed flex items-start gap-2.5">
			<Sparkles class="w-4 h-4 text-indigo-400 shrink-0 mt-0.5" />
			<span>{feedbackText}</span>
		</div>
	{/if}
</div>
