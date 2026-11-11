package database

import (
	"database/sql"
	"strings"
)

// SeedData inserts standard languages, sample presets, and common vocabulary
func SeedData(db *sql.DB) error {
	// 1. Languages
	languages := []struct {
		code, name, nativeName, flag, voice string
		pitch, rate                         float64
	}{
		{"en-US", "English (US)", "English (US)", "🇺🇸", "en-US", 1.0, 0.95},
		{"en-GB", "English (UK)", "English (UK)", "🇬🇧", "en-GB", 1.0, 0.95},
		{"es-ES", "Spanish (Spain)", "Español", "🇪🇸", "es-ES", 1.0, 0.95},
		{"fr-FR", "French (France)", "Français", "🇫🇷", "fr-FR", 1.0, 0.95},
		{"de-DE", "German", "Deutsch", "🇩🇪", "de-DE", 1.0, 0.95},
		{"it-IT", "Italian", "Italiano", "🇮🇹", "it-IT", 1.0, 0.95},
		{"ja-JP", "Japanese", "日本語", "🇯🇵", "ja-JP", 1.0, 1.0},
	}

	for _, l := range languages {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO languages (code, name, native_name, flag, voice_name, pitch, rate)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			l.code, l.name, l.nativeName, l.flag, l.voice, l.pitch, l.rate)
		if err != nil {
			return err
		}
	}

	// 2. Presets (Learning Texts)
	presets := []struct {
		id, langCode, category, title, description, content, difficulty string
	}{
		{
			id:          "en-daily-1",
			langCode:    "en-US",
			category:    "Daily Conversation",
			title:       "Ordering Specialty Coffee in Seattle",
			description: "Practical conversation focusing on natural rhythm, reduction, and polite requests.",
			content:     "Hi there! Could I get an oat milk flat white with an extra shot? Also, do you happen to have any fresh almond croissants left today?",
			difficulty:  "Beginner",
		},
		{
			id:          "en-tech-1",
			langCode:    "en-US",
			category:    "Tech & Architecture",
			title:       "Introducing Distributed Microservices",
			description: "Clear presentation cadence with technical terminology and linking sounds.",
			content:     "Good morning everyone. Today, I would like to walk you through our architectural migration towards asynchronous event-driven services. Notice how decoupling the message pipeline significantly minimizes latency under peak workloads.",
			difficulty:  "Intermediate",
		},
		{
			id:          "en-intonation-1",
			langCode:    "en-US",
			category:    "Phonetics & Intonation",
			title:       "Liaison & Rhythm Training",
			description: "Designed specifically to practice vowel-to-vowel linking, flapping 't', and thought groups.",
			content:     "Did you ask about it? I thought it was out of stock, but an employee found one at the back of the store.",
			difficulty:  "Advanced",
		},
		{
			id:          "es-daily-1",
			langCode:    "es-ES",
			category:    "Daily Conversation",
			title:       "En un restaurante en Barcelona",
			description: "Conversación cotidiana para practicar el ritmo silábico del español.",
			content:     "¡Buenas tardes! ¿Tienen una mesa libre para dos personas en la terraza? Nos gustaría probar las tapas típicas y una jarra de sangría.",
			difficulty:  "Beginner",
		},
		{
			id:          "fr-daily-1",
			langCode:    "fr-FR",
			category:    "Daily Conversation",
			title:       "Une promenade matinale à Paris",
			description: "Pratiquez les liaisons et l'enchaînement vocalique de la langue française.",
			content:     "Bonjour ! Je cherche la station de métro la plus proche d'ici. Est-ce que c'est loin à pied, ou vaut-il mieux prendre un bus ?",
			difficulty:  "Intermediate",
		},
		{
			id:          "de-daily-1",
			langCode:    "de-DE",
			category:    "Daily Conversation",
			title:       "In einem Café in Berlin",
			description: "Praktische Konversation mit klaren Konsonanten und zusammengesetzten Substantiven.",
			content:     "Guten Tag! Ich hätte gerne einen Cappuccino mit Hafermilch und ein Stück Apfelstrudel, bitte. Kann man hier mit Karte bezahlen?",
			difficulty:  "Beginner",
		},
		{
			id:          "it-daily-1",
			langCode:    "it-IT",
			category:    "Daily Conversation",
			title:       "Al bar a Roma",
			description: "Conversazione tipica al bar italiano: ritmo melodico e pronuncia chiara delle doppie consonanti.",
			content:     "Buongiorno! Vorrei un caffè espresso al banco e un cornetto al pistacchio, per favore. Quant'è in tutto?",
			difficulty:  "Beginner",
		},
	}

	for _, p := range presets {
		_, err := db.Exec(`
			INSERT OR REPLACE INTO presets (id, lang_code, category, title, description, content, difficulty)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			p.id, p.langCode, p.category, p.title, p.description, p.content, p.difficulty)
		if err != nil {
			return err
		}
	}

	// 3. Vocabularies (Sample IPA & definitions)
	vocabularies := []struct {
		langCode, word, ipa, pos, meaning, example, exampleTr string
	}{
		// English words
		{"en-US", "comfortable", "/ˈkʌmftəbl/", "adjective", "快適な、心地よい", "This ergonomic chair is very comfortable.", "この人間工学チェアはとても快適です。"},
		{"en-US", "architecture", "/ˈɑːrkɪtektʃər/", "noun", "建築、基本設計、アーキテクチャ", "Clean architecture promotes maintainability.", "クリーンアーキテクチャは保守性を高めます。"},
		{"en-US", "asynchronous", "/eɪˈsɪŋkrənəs/", "adjective", "非同期の", "We use asynchronous message queues.", "非同期メッセージキューを採用しています。"},
		{"en-US", "decoupling", "/diːˈkʌplɪŋ/", "noun", "疎結合化、分離", "Decoupling services improves scalability.", "サービスの疎結合化によって拡張性が向上します。"},
		{"en-US", "latency", "/ˈleɪtnsi/", "noun", "待ち時間、遅延", "Low latency is critical for real-time speech.", "低遅延はリアルタイム音声において不可欠です。"},
		{"en-US", "croissant", "/kwɑːˈsɑːŋ/", "noun", "クロワッサン", "A warm buttery croissant.", "温かいバターたっぷりのクロワッサン。"},
		{"en-US", "migration", "/maɪˈɡreɪʃn/", "noun", "移行、マイグレーション", "The database migration completed smoothly.", "データベース移行は円滑に完了しました。"},
		{"en-US", "employee", "/ɪmˈplɔɪiː/", "noun", "従業員、店員", "An employee assisted us.", "店員さんが私たちを案内してくれました。"},
		{"en-US", "specialty", "/ˈspeʃəlti/", "noun / adj", "名物、特製、専門", "Specialty coffee brewed with care.", "丁寧に淹れられたスペシャルティコーヒー。"},
		{"en-US", "almond", "/ˈɑːmənd/", "noun", "アーモンド（'l'を発音しない点に注意）", "Almond milk is a healthy alternative.", "アーモンドミルクは健康的な選択肢です。"},
		{"en-US", "pipeline", "/ˈpaɪplaɪn/", "noun", "パイプライン、一連の処理経路", "The CI/CD pipeline runs tests automatically.", "CI/CDパイプラインがテストを自動実行します。"},
		{"en-US", "flapping", "/ˈflæpɪŋ/", "noun", "（音声学）弾き音化（waterがワラーのようになる現象）", "Flapping 't' occurs between vowels.", "母音に挟まれたtは弾き音化（フラッピング）が起きます。"},
		
		// Spanish words
		{"es-ES", "restaurante", "/restawˈrante/", "sustantivo", "レストラン", "¿Hay algún restaurante cerca?", "近くにレストランはありますか？"},
		{"es-ES", "terraza", "/teˈraθa/", "sustantivo", "テラス席", "Prefiero sentarme en la terraza.", "テラス席に座りたいです。"},
		{"es-ES", "sangría", "/saŋˈɡri.a/", "sustantivo", "サングリア（伝統的なワインカクテル）", "Una copa de sangría fría.", "冷たいサングリア一杯。"},

		// French words
		{"fr-FR", "promenade", "/pʁɔm.nad/", "nom", "散歩、散策", "Une promenade le long de la Seine.", "セーヌ川沿いの散歩。"},
		{"fr-FR", "station", "/sta.sjɔ̃/", "nom", "駅、停留所", "La prochaine station de métro.", "次の地下鉄の駅。"},
		{"fr-FR", "liaison", "/ljɛ.zɔ̃/", "nom", "（音声学）リエゾン、音の連結", "La liaison en français est essentielle.", "フランス語におけるリエゾンは不可欠です。"},

		// German words
		{"de-DE", "hafermilch", "/ˈhaːfɐˌmɪlç/", "Substantiv", "オーツミルク", "Kaffee mit Hafermilch.", "オーツミルク入りコーヒー。"},
		{"de-DE", "apfelstrudel", "/ˈapfl̩ˌʃtʁuːdl̩/", "Substantiv", "アップルシュトゥルーデル（オーストリア・ドイツ伝統のリンゴパイ）", "Ein warmes Stück Apfelstrudel.", "温かい一切れのアップルパイ。"},
		{"de-DE", "aussprache", "/ˈaʊ̯sˌʃpʁaːxə/", "Substantiv", "発音", "Eine klare deutsche Aussprache.", "明瞭なドイツ語の発音。"},
		{"de-DE", "rechnung", "/ˈʁɛçnʊŋ/", "Substantiv", "お会計、請求書", "Die Rechnung bitte!", "お会計をお願いします！"},

		// Italian words
		{"it-IT", "espresso", "/eˈsprɛs.so/", "sostantivo", "エスプレッソコーヒー", "Un caffè espresso, grazie.", "エスプレッソを1杯お願いします。"},
		{"it-IT", "cornetto", "/korˈnet.to/", "sostantivo", "コルネット（イタリア風クロワッサン）", "Un cornetto caldo al mattino.", "朝の温かいコルネット。"},
		{"it-IT", "pistacchio", "/piˈstak.kjo/", "sostantivo", "ピスタチオ", "Gelato al pistacchio di Bronte.", "ブロンテ産ピスタチオのジェラート。"},
		{"it-IT", "piacere", "/pjaˈtʃe.re/", "sostantivo", "はじめまして、喜び", "Molto piacere di conoscerla!", "お会いできて光栄です！"},
		{"it-IT", "pronuncia", "/proˈnun.tʃa/", "sostantivo", "発音", "La corretta pronuncia italiana.", "正しいイタリア語の発音。"},
	}

	for _, v := range vocabularies {
		cleanWord := strings.ToLower(strings.Trim(v.word, " .,!?\"'"))
		_, err := db.Exec(`
			INSERT OR REPLACE INTO vocabularies (lang_code, word, clean_word, ipa, pos, meaning, example, example_translation)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			v.langCode, v.word, cleanWord, v.ipa, v.pos, v.meaning, v.example, v.exampleTr)
		if err != nil {
			return err
		}
	}

	return nil
}

