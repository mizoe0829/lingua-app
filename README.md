# LinguaLight — 多言語インタラクティブ学習 & 音声発音ワークベンチ

[![Svelte 5](https://img.shields.io/badge/Frontend-Svelte%205%20(Runes)-FF3E00.svg)](https://svelte.dev)
[![SvelteKit](https://img.shields.io/badge/Metaframework-SvelteKit%202-FF3E00.svg)](https://kit.svelte.dev)
[![Go 1.24](https://img.shields.io/badge/Backend-Go%201.24-00ADD8.svg)](https://golang.org)
[![SQLite](https://img.shields.io/badge/Database-SQLite%20(WAL%20Mode)-003B57.svg)](https://sqlite.org)
[![Gemini API](https://img.shields.io/badge/AI-Google%20Gemini%20Flash-4285F4.svg)](https://ai.google.dev)
[![Docker](https://img.shields.io/badge/Container-Docker%20Compose-2496ED.svg)](https://docker.com)

---

## 📖 プロジェクト概要 (Overview)

**LinguaLight** は、英語をはじめとする多言語（スペイン語、フランス語、ドイツ語、日本語等）の「実際に話す・聞く力」を 0 秒のレスポンスで鍛えるためのインタラクティブ語学ワークベンチです。

土地の事情や環境によって高度な言語学習スクールやネイティブ対話環境が身近に整っていない状況でも、**「手元のテキストを流し込むだけで一瞬で単語に分解し、0msで発音させ、AIがリアルタイムにイントネーションや音の繋がり（リエゾン）を解説する」** 自律的な学習体験を提供します。

---

## 💡 設計思想: "Write, Right, Light"

本プロダクトは、作者の一貫した開発信条である **"Write, Right, Light"** に基づいて設計・実装されています。

```mermaid
flowchart TD
    subgraph Browser ["Frontend: SvelteKit 2 + Svelte 5 (Port 5173)"]
        UIInput["学習テキスト入力 / プリセット選択"]
        TokenBoard["インタラクティブ・トークンボード<br/>(単語分割・ホバー辞書・0ms発音)"]
        AudioEngine["Web Speech API コントローラー<br/>(Runes: $state / $effect 音声同期ハイライト)"]
        AIPanel["AI イントネーション & 構文解説<br/>(SSE ストリーミング描画)"]
    end

    subgraph GoBackend ["Backend: Go 1.24 (Port 8085, ~17MB Binary)"]
        Mux["標準 net/http ルーター & CORS"]
        Tokenizer["多言語トークナイザー & リエゾン判定"]
        GeminiSvc["Gemini API (3.8 / 2.5 Flash)<br/>+ ローカルルールベース・フォールバック"]
        DB["SQLite 3 (WALモード, modernc Pure Go)"]
    end

    UIInput -->|POST /api/tokenize| Tokenizer
    Tokenizer -->|IPA & 品詞マッピング| TokenBoard
    TokenBoard -->|単語タップ: 0ms| AudioEngine
    AudioEngine -->|boundary イベント| TokenBoard
    TokenBoard -->|文単位のAI解説要求| GeminiSvc
    GeminiSvc -->|text/event-stream (SSE)| AIPanel
    TokenBoard -->|単語帳保存 /api/bookmarks| DB
```

### 1. Light（超低遅延・極小フットプリント）
- **Svelte 5 (Runes)**: 
  - 仮想DOMのdiff検出オーバーヘッドを完全に排除。シグナルベースのリアクティビティ（`$state`, `$derived`, `$effect`）により、数百単語の長文でもホバーや音声再生中のカラオケ風ハイライトが 60fps（0ms遅延）で追従。
- **Go 1.24 シングルバイナリ**: 
  - CGO不要の Pure Go SQLite (`modernc.org/sqlite`) を採用し、バイナリサイズわずか **17MB**。
  - メモリ消費量は 20MB 台、コールドスタート数ミリ秒で即座にリクエストを捌きます。

### 2. Right（ドメインの本質・データ整合性・高可用性）
- **多言語拡張ドメインモデル**: 
  - 言語コード（ISO 639-1）、IPA国際音声記号、品詞（POS）、音の繋がり（Liaison）を厳格にモデル化。
- **2重のフォールバック・冗長化設計**: 
  - Gemini API キーが未設定、クォータ制限、または通信障害が発生した場合でも、Go内部の**ルールベース解析エンジン**が即座に代替ストリーミングを実行。
  - 音声再生も外部API依存を排除し、ブラウザ標準の **Web Speech API** を活用することで完全オフラインでも発音学習を継続可能。

### 3. Write（型安全性・品質担保・明文化）
- フロントエンド（TypeScript + Svelte 5 型定義）とバックエンド（Go 構造体）のスキーマを完全一致。
- `go test` によるトークナイザー分割精度・リエゾン検出・DB初期化の自動テスト網。
- `svelte-check` による静的型検査（エラー・警告ゼロ）。

---

## 🚀 主要機能 (Features)

| 機能 | 詳細 |
| :--- | :--- |
| **0ms インタラクティブ発音** | 単語をクリックした瞬間に Web Speech API でネイティブ発音を再生。 |
| **IPA国際音声記号 & 辞書ポップアップ** | 単語ホバー時に、IPA発音記号・品詞・日本語訳・リエゾンヒントを瞬時にプレビュー。 |
| **リアルタイム・カラオケ同期ハイライト** | 音声読み上げイベント（`boundary`）を検知し、発声中の単語をミリ秒精度でハイライト追従。 |
| **AI イントネーション & 構文解析 (SSE)** | Gemini 3.8/2.5 Flash により、「イントネーションの山」「音の脱落（リダクション）」「構文の骨組み」をリアルタイムにタイプライター風ストリーミング表示。 |
| **多言語切り替え** | 英語（US/UK）、スペイン語、フランス語、ドイツ語、日本語などをシームレスに切り替え。 |
| **ワンタップ学習プリセット** | 日常会話、IT技術プレゼン、音声学・リエゾン特訓などのサンプル文を即座に読み込み。 |
| **マイスペース（単語帳・フレーズ保存）** | 気になった単語や文をワンクリックでSQLiteに保存・復習可能。 |

---

## 🛠️ 技術スタック (Tech Stack)

### Frontend
- **Framework**: SvelteKit 2 (`@sveltejs/kit` ^2.63.0)
- **UI Library**: Svelte 5 (`svelte` ^5.56.1, Runes: `$state`, `$derived`, `$effect`)
- **Language**: TypeScript (`^6.0.3`)
- **Styling**: Tailwind CSS v3 (`^3.4.17`), PostCSS, Autoprefixer
- **Icons**: Lucide Svelte (`^0.475.0`)
- **Speech**: Web Speech API (`SpeechSynthesisUtterance`)

### Backend
- **Language**: Go 1.24 (Windows / Linux)
- **Server**: Go 標準 `net/http` (SSE `text/event-stream` サポート)
- **Database**: SQLite 3 (WALモード, `modernc.org/sqlite` Pure Go)
- **AI/LLM**: Google Gemini API (`gemini-2.5-flash` / `gemini-3.8-flash`)

### DevOps & Infrastructure
- **Container**: Docker / Docker Compose (Multi-stage Alpine builds)

---

## 🏃 起動方法 (Getting Started)

### 方法 1: Docker Compose で一括起動（推奨）

```sh
# リポジトリのクローン
git clone https://github.com/mizoe0829/lingua-app.git
cd lingua-app

# （任意）Gemini APIキーを設定する場合は環境変数を渡す
export GEMINI_API_KEY="your-api-key-here"

# コンテナのビルドと起動
docker compose up --build
```
- フロントエンド: `http://localhost:5173`
- バックエンド API: `http://localhost:8085`

---

### 方法 2: ローカル環境で起動

#### 1. バックエンド (Go)
```sh
cd backend
go run cmd/server/main.go
# 起動確認: http://localhost:8085/health
```

#### 2. フロントエンド (SvelteKit)
```sh
cd frontend
npm install
npm run dev
# アクセス: http://localhost:5173
```

---

## 🧪 テストの実行 (Testing)

### Go バックエンド単体テスト
```sh
cd backend
go test -v ./...
```

### フロントエンド型検査 (TypeScript & Svelte 5)
```sh
cd frontend
npm run check
```

