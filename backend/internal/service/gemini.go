package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"lingua-backend/internal/model"
)

type GeminiService struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiService() *GeminiService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" || modelName == "gemini-2.5-flash" {
		modelName = "gemini-3.8-flash"
	}

	dialer := &net.Dialer{
		Timeout:   3 * time.Second,
		DualStack: false,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Enforce IPv4 (tcp4) to eliminate Windows dual-stack IPv6 DNS hang (speeds up from 75s to 0.3s)
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &GeminiService{
		apiKey: apiKey,
		model:  modelName,
		client: &http.Client{
			Transport: transport,
			Timeout:   8 * time.Second,
		},
	}
}

// StreamPhoneticAnalysis streams speech, intonation, and grammar insights via SSE
func (g *GeminiService) StreamPhoneticAnalysis(ctx context.Context, req model.AnalyzeRequest, flusher http.Flusher, w http.ResponseWriter) error {
	// If API key is missing, seamlessly fall back to local rule-based analysis
	if g.apiKey == "" {
		return g.streamFallbackAnalysis(ctx, req, flusher, w)
	}

	prompt := g.buildAnalysisPrompt(req)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", g.model, g.apiKey)

	bodyData := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"topP":            0.8,
			"maxOutputTokens": 1024,
		},
	}

	jsonBytes, err := json.Marshal(bodyData)
	if err != nil {
		return g.streamFallbackAnalysis(ctx, req, flusher, w)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return g.streamFallbackAnalysis(ctx, req, flusher, w)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return g.streamFallbackAnalysis(ctx, req, flusher, w)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data:")
		jsonData = strings.TrimSpace(jsonData)
		if jsonData == "" || jsonData == "[DONE]" {
			continue
		}

		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(jsonData), &geminiResp); err == nil {
			if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
				chunkText := geminiResp.Candidates[0].Content.Parts[0].Text
				payload, _ := json.Marshal(map[string]string{"chunk": chunkText})
				fmt.Fprintf(w, "data: %s\n\n", payload)
				flusher.Flush()
			}
		}
	}

	// Send complete marker
	fmt.Fprintf(w, "data: {\"done\": true}\n\n")
	flusher.Flush()
	return nil
}

func (g *GeminiService) buildAnalysisPrompt(req model.AnalyzeRequest) string {
	target := req.Sentence
	if req.Word != "" {
		target = fmt.Sprintf("対象単語: '%s' (文中: \"%s\")", req.Word, req.Sentence)
	}

	return fmt.Sprintf(`あなたは語学教育・音声学（Phonetics & Intonation）のエキスパートAIです。
対象言語: %s
学習対象: %s

以下の4項目について、学習者が実際に「声に出して話す・聞く」際に最も役立つように、Markdown形式で分かりやすく解説してください：

1. 🗣️ **発音と音の変化 (Phonetic Features & Linking)**:
   - IPA発音記号、母音・子音の調音ポイント
   - リエゾン（音の繋がり）、リダクション（音の脱落・弱化）、フラッピングなどの音声現象
2. 📈 **イントネーションとリズム (Intonation & Stress Rhythm)**:
   - どこに強勢（ピッチ・音量）を置き、どこを弱く速く発声するか
   - 息継ぎ（ポーズ）を入れるべき区切り（Sense Group / Thought Group）
3. 🧩 **構文と語順ニュアンス (Syntax & Structure)**:
   - 誰が・どうする・何を、修飾関係の骨組み
4. 💡 **ネイティブが話す時の実践アドバイス**:
   - 会話で自然に聞こえるワンポイントアドバイス
`, req.LangCode, target)
}

func (g *GeminiService) streamFallbackAnalysis(ctx context.Context, req model.AnalyzeRequest, flusher http.Flusher, w http.ResponseWriter) error {
	fallbackText := fmt.Sprintf(`### 🗣️ 音声・イントネーション解説（ローカル高速解析モード）

**対象文**: *"%s"* (%s)

#### 1. 📈 イントネーション & リズムの山
- **強勢（ストレス）**: 意味を持つ主要な内容語（名詞・動詞・形容詞）にピッチ（音の高さ）の山を作ります。
- **機能語の弱化**: 前置詞・冠詞・助動詞は弱く短く発声し、英語特有の「強弱リズム」を意識してください。
- **息継ぎ（ポーズ）**: 意味のまとまり（チャンク）ごとに微小な休止を挟むと、ネイティブにとって聞き取りやすくなります。

#### 2. 🔗 音の連結（リンキング・リエゾン）
- 子音で終わる単語の直後に母音で始まる単語が続く場合、音が繋がってひとつの単語のように発音されます。

#### 3. 💡 スピーキングのコツ
- 口の開きを大きく保ち、腹式呼吸で息をしっかり前に押し出すことで、明瞭な響きになります。
- 画面の「再生」ボタンを押して、ハイライトに合わせたシャドーイング（後について発話）を繰り返しましょう！
`, req.Sentence, req.LangCode)

	lines := strings.Split(fallbackText, "\n")
	for _, line := range lines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, _ := json.Marshal(map[string]string{"chunk": line + "\n"})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
		time.Sleep(30 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: {\"done\": true}\n\n")
	flusher.Flush()
	return nil
}

// StreamDialog handles conversational sparring with difficulty levels and translation
func (g *GeminiService) StreamDialog(ctx context.Context, req model.DialogStreamRequest, flusher http.Flusher, w http.ResponseWriter) error {
	if g.apiKey == "" {
		// Log informative fallback message
		fmt.Printf("[Dialog] GEMINI_API_KEY is not set. Using dynamic intelligent local dialog engine for: '%s'\n", req.UserMessage)
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}

	prompt := g.buildDialogPrompt(req)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", g.model, g.apiKey)

	bodyData := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.6,
			"topP":            0.9,
			"maxOutputTokens": 1024,
		},
	}

	jsonBytes, err := json.Marshal(bodyData)
	if err != nil {
		fmt.Printf("[Dialog] Error marshaling body: %v\n", err)
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Printf("[Dialog] Error creating request: %v\n", err)
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		fmt.Printf("[Dialog] Network/API error: %v. Falling back to local engine.\n", err)
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Dialog] Gemini API responded with status %d: %s. Falling back to local engine.\n", resp.StatusCode, string(respBody))
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}

	reader := bufio.NewReader(resp.Body)
	currentSection := "reply"

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data:")
		jsonData = strings.TrimSpace(jsonData)
		if jsonData == "" || jsonData == "[DONE]" {
			continue
		}

		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(jsonData), &geminiResp); err == nil {
			if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
				chunkText := geminiResp.Candidates[0].Content.Parts[0].Text

				// Section transitions
				if strings.Contains(chunkText, "[TRANSLATION]") {
					parts := strings.Split(chunkText, "[TRANSLATION]")
					g.sendDialogChunk(w, flusher, currentSection, parts[0])
					currentSection = "translation"
					if len(parts) > 1 {
						g.sendDialogChunk(w, flusher, currentSection, parts[1])
					}
					continue
				} else if strings.Contains(chunkText, "[BETTER_PHRASING]") {
					parts := strings.Split(chunkText, "[BETTER_PHRASING]")
					g.sendDialogChunk(w, flusher, currentSection, parts[0])
					currentSection = "better_phrasing"
					if len(parts) > 1 {
						g.sendDialogChunk(w, flusher, currentSection, parts[1])
					}
					continue
				} else if strings.Contains(chunkText, "[TIP]") {
					parts := strings.Split(chunkText, "[TIP]")
					g.sendDialogChunk(w, flusher, currentSection, parts[0])
					currentSection = "tip"
					if len(parts) > 1 {
						g.sendDialogChunk(w, flusher, currentSection, parts[1])
					}
					continue
				}

				clean := strings.ReplaceAll(chunkText, "[REPLY]", "")
				g.sendDialogChunk(w, flusher, currentSection, clean)
			}
		}
	}

	donePayload, _ := json.Marshal(model.DialogStreamChunk{Type: "done"})
	fmt.Fprintf(w, "data: %s\n\n", donePayload)
	flusher.Flush()
	return nil
}

func (g *GeminiService) sendDialogChunk(w http.ResponseWriter, flusher http.Flusher, section, text string) {
	if text == "" {
		return
	}
	payload, _ := json.Marshal(model.DialogStreamChunk{
		Type:  section,
		Chunk: text,
	})
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}

func (g *GeminiService) buildDialogPrompt(req model.DialogStreamRequest) string {
	historyStr := ""
	for _, m := range req.History {
		historyStr += fmt.Sprintf("- %s: %s\n", m.Role, m.Content)
	}

	levelGuide := ""
	switch strings.ToLower(req.Level) {
	case "beginner":
		levelGuide = "初級（A1-A2）。短くシンプルな単語で構成し、1〜2文でやさしく返答してください。質問もシンプルに。"
	case "advanced":
		levelGuide = "上級（C1-C2）。洗練された成句・複文を用い、自然で少し速い議論やトピックの掘り下げを行ってください。"
	default:
		levelGuide = "中級（B1-B2）。日常会話やビジネスで最もよく使われる表現を用い、自然な会話のキャッチボール（2〜3文）をしてください。"
	}

	return fmt.Sprintf(`あなたは語学学習者のスピーキング・壁打ち対話パートナーです。
対象言語: %s
学習者の習熟度レベル: %s (%s)

【過去の対話履歴】
%s

【学習者の最新の発話】
"%s"

以下の4つのタグで区切って出力してください。タグ以外の前置きや挨拶は一切含めないでください。

[REPLY]
（対象言語での返答。会話がテンポ良く続くように質問も添えてください）
[TRANSLATION]
（返答の自然な日本語訳）
[BETTER_PHRASING]
（学習者の発話「%s」をもっとネイティブらしく自然にする言い換え提案 1〜2選）
[TIP]
（発音・イントネーション・語法に関するワンポイント助言）
`, req.LangCode, req.Level, levelGuide, historyStr, req.UserMessage, req.UserMessage)
}

func normalizeText(s string) string {
	lower := strings.ToLower(s)
	// Normalize common accented vowels (e.g. café -> cafe)
	replacer := strings.NewReplacer(
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"á", "a", "à", "a", "â", "a", "ä", "a",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ñ", "n", "ç", "c",
	)
	return replacer.Replace(lower)
}

func hasWord(s, target string) bool {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, w := range words {
		if strings.EqualFold(w, target) {
			return true
		}
	}
	return false
}

func (g *GeminiService) streamFallbackDialog(ctx context.Context, req model.DialogStreamRequest, flusher http.Flusher, w http.ResponseWriter) error {
	norm := normalizeText(req.UserMessage)
	level := strings.ToLower(req.Level)

	reply := ""
	trans := ""
	better := ""
	tip := ""

	isRecommendation := strings.Contains(norm, "recommend") || strings.Contains(norm, "suggest") || strings.Contains(norm, "where can") || strings.Contains(norm, "any good")
	isThirsty := hasWord(norm, "thirsty") || hasWord(norm, "drink") || hasWord(norm, "drinks") || strings.Contains(norm, "something to drink")

	// 1. Food / Cafe / Thirsty / Dining intent (Checked first to avoid any substring overlaps)
	if strings.Contains(norm, "coffee") || strings.Contains(norm, "cafe") || strings.Contains(norm, "croissant") || strings.Contains(norm, "restaurant") || strings.Contains(norm, "bakery") || strings.Contains(norm, "tea") || hasWord(norm, "food") || hasWord(norm, "eat") || hasWord(norm, "lunch") || hasWord(norm, "dinner") || isThirsty {
		if isThirsty {
			switch level {
			case "beginner":
				reply = "You should get a cold drink right away! Would you like an iced tea, fresh lemonade, or a cold water?"
				trans = "すぐに冷たいものを飲んで喉を潤しましょう！アイスティー、フレッシュレモネード、それとも冷たいお水はいかがですか？"
				better = "「That sounds nice. I'm thirsty.」に加えて「That sounds great, I could really use a cold drink right now!」と言うとさらに自然で感情が伝わります。"
				tip = "「thirsty」の [θ] 音は舌先を上前歯の先端に軽く当てて息を抜きます。[s]（サー）にならないように意識しましょう。"
			case "advanced":
				reply = "There's nothing quite like a crisp, refreshing beverage when you're parched. That café has iced sparkling cold brew with lemon that is absolute perfection. Shall we head over?"
				trans = "喉がカラカラの時に飲む爽快なドリンクに勝るものはありませんよね。あのカフェにはレモンを添えたスパークリング・コールドブリューがあって、これが絶品なんです。行ってみませんか？"
				better = "「I'm thirsty」の上級表現として「I'm pretty parched」（喉が乾ききっている）を使うとネイティブらしいボキャブラリーになります。"
				tip = "「parched」の [tʃt] は息を破裂させてクリアに発音し、語尾を曖昧にしないのがコツです。"
			default:
				reply = "Oh, you must get hydrated! That café makes fantastic iced fruit teas and sparkling sodas. What kind of drink are you craving?"
				trans = "おや、水分補給しないといけませんね！そのカフェはアイスフルーツティーや炭酸ソーダがとても美味しいですよ。どんな飲み物を欲していますか？"
				better = "「I'm craving a cold drink」（冷たいものが欲しくてたまらない）という表現も日常会話で非常によく使われます。"
				tip = "「craving」は [kreɪ-vɪŋ] で、[v] は上前歯を下唇に軽く触れさせて振動させます。"
			}
		} else if isRecommendation {
			switch level {
			case "beginner":
				reply = "There's a wonderful café called 'Green Terrace' just around the corner! Their iced latte and pastries are amazing. Would you like to check it out?"
				trans = "すぐ角を曲がったところに『グリーン・テラス』という素敵なカフェがありますよ！アイスラテとペストリーが絶品です。行ってみますか？"
				better = "「Could you recommend a good café?」の他にも「Do you know any nice cafés nearby?」と聞くとフレンドリーで自然です。"
				tip = "「café」は [kæˈfeɪ] と語尾の母音 [eɪ] にピッチを上げ気味にアクセントを置きます。"
			case "advanced":
				reply = "If you're into specialty coffee, you should definitely check out 'The Roasted Bean' two blocks away. They do incredible single-origin pourovers and the atmosphere is quite serene. Are you looking to grab a quick cup or sit down to work?"
				trans = "スペシャルティコーヒーがお好きなら、2ブロック先の『The Roasted Bean』が断然おすすめです。素晴らしいシングルオリジンのハンドドリップが楽しめ、雰囲気もとても落ち着いていますよ。テイクアウトですか、それともゆっくり作業されますか？"
				better = "「Are there any local hidden gems for coffee?」のように「hidden gems」（隠れた名店）を使うと洗練された表現になります。"
				tip = "「two blocks away」の [ks-ə] はリエゾン（音の連結）を意識して [tʰuː blɑksəweɪ] と滑らかに繋げましょう。"
			default: // intermediate
				reply = "I'd love to! There's a cozy café called 'Artisan Brew' about five minutes walk from here. They have great roasted coffee and free Wi-Fi. Do you prefer espresso drinks or drip coffee?"
				trans = "喜んで！ここから徒歩5分ほどの場所に『Artisan Brew』という居心地の良いカフェがありますよ。焙煎コーヒーが美味しくWi-Fiも使えます。エスプレッソ系とドリップコーヒー、どちらがお好みですか？"
				better = "「Can you suggest a good place for coffee?」や「What's your favorite café around here?」も非常によく使われる自然な表現です。"
				tip = "「about five minutes walk」は [ə-baʊt faɪv mɪnɪts wɔːk] と一息のリズムで流れるように発音します。"
			}
		} else {
			// Ordering or general café talk
			switch level {
			case "beginner":
				reply = "Sure! We have hot coffee, iced latte, and fresh tea. Which size would you like?"
				trans = "かしこまりました！ホットコーヒー、アイスラテ、淹れたてのお茶がございます。サイズはどちらになさいますか？"
				better = "注文時は「I want...」よりも「Could I have a coffee, please?」と頼むと大変丁寧です。"
				tip = "「Which size would you like」の文末は、選択肢を問う疑問文なので音を下げて発音します。"
			case "advanced":
				reply = "Certainly! Today's single-origin batch brew features floral Ethiopian beans with notes of jasmine and citrus. Would you care for whole milk, or a plant-based alternative?"
				trans = "かしこまりました！本日のシングルオリジンはジャスミンと柑橘の香るエチオピア産です。牛乳になさいますか？それとも植物性ミルクがよろしいですか？"
				better = "「Could I get that with an extra shot?」や「Can you make that with oat milk?」とサラッと言えると通っぽいです。"
				tip = "「Would you care for」は非常に洗練された接客英語です。[wʊ-dʒu-kɛər-fər] と連結します。"
			default:
				reply = "Of course! Coming right up. For here, or to go? We also have fresh pastries on the counter today."
				trans = "もちろん！すぐにお作りしますね。店内ですか、それともお持ち帰りですか？カウンターには焼きたてのペストリーもありますよ。"
				better = "「For here or to go?」（店内か持ち帰りか）は海外カフェの頻出フレーズです。「To go, please!」と即答できるようにしましょう。"
				tip = "「For here」と「to go」の対比では、それぞれのキーワードにピッチのアクセントを置きます。"
			}
		}
	} else if hasWord(norm, "hello") || hasWord(norm, "hi") || hasWord(norm, "hey") || hasWord(norm, "morning") || hasWord(norm, "hola") || hasWord(norm, "bonjour") || hasWord(norm, "ciao") || hasWord(norm, "hallo") {
		// 2. Greeting intent (strict standalone whole word matching - never match 'hi' in 'thirsty' or 'this')
		switch level {
		case "beginner":
			reply = "Hello! Great to meet you. How are you doing today?"
			trans = "こんにちは！お会いできて嬉しいです。今日の調子はいかがですか？"
			better = "「Hi there!」や「Good morning!」と明るくピッチを上げて言うと、ネイティブにとても親しみやすい印象を与えます。"
			tip = "英語の挨拶は語尾の音を軽くキュッと上げると（Up-gliding）、フレンドリーで聞き取りやすくなります。"
		case "advanced":
			reply = "Hey, good to see you! How has your week been treating you so far?"
			trans = "やあ、会えて嬉しいよ！今週の調子はどう？順調に進んでる？"
			better = "親しい同僚や友人には「How's everything going?」や「What's new?」とフランクに切り出すのが一般的です。"
			tip = "「treating you」の部分は [t] がリエゾンして滑らかに流れるように発音するとネイティブらしいです。"
		default: // intermediate
			reply = "Hello there! Nice to talk to you. What are you up to today?"
			trans = "こんにちは！お話しできて光栄です。今日はどんな予定があるんですか？"
			better = "「Nice to meet you」は初対面、「Nice to see you」は再会時に使い分けると自然です。"
			tip = "「What are you up to」は [wʌt-ər-ju-ʌp-tə] とひと繋がりのリズムで一気に発声しましょう。"
		}
	} else if hasWord(norm, "code") || hasWord(norm, "coding") || hasWord(norm, "programming") || hasWord(norm, "software") || hasWord(norm, "golang") || (hasWord(norm, "go") && (hasWord(norm, "lang") || hasWord(norm, "concurrency") || hasWord(norm, "routine"))) || hasWord(norm, "svelte") || hasWord(norm, "database") || hasWord(norm, "api") || hasWord(norm, "refactor") {
		// 3. Tech / Engineering intent (strict whole-word matching, never matching substring 'go' in 'good')
		switch level {
		case "beginner":
			reply = "Programming is awesome! What kind of software are you building right now?"
			trans = "プログラミングは素晴らしいですね！今はどんなソフトウェアを作っているのですか？"
			better = "「I am writing code」よりも「I'm working on a web app」や「I build backend services」と表現すると自然です。"
			tip = "「building」の [l] 音は舌先を上前歯の付け根にしっかり押し当てて響かせます。"
		case "advanced":
			reply = "Decoupling distributed microservices is always a tricky trade-off between latency and data consistency. How are you managing state sync across nodes?"
			trans = "分散マイクロサービスの疎結合化は、常に低遅延とデータ整合性のトレードオフが悩ましいですよね。ノード間の状態同期はどう対処されていますか？"
			better = "技術ディスカッションでは「We decided to adopt... because of...」と選定理由を明示すると説得力が増します。"
			tip = "「microservices」や「consistency」などの多音節テクニカル用語は、第1・第2ストレスの位置を意識してハキハキ発声します。"
		default:
			reply = "That sounds like a fascinating project! Are you focusing on backend performance or frontend reactivity?"
			trans = "とても面白そうなプロジェクトですね！バックエンドのパフォーマンスとフロントエンドのリアクティビティ、どちらに重点を置いていますか？"
			better = "「I like Go」だけでなく「Go's concurrency model really helps reduce latency」のように理由を添えてみましょう。"
			tip = "「backend performance」と語句が続くときは、単語と単語の間に余計な母音 [u] を挟まないように注意します。"
		}
	} else if strings.Contains(norm, "learn") || strings.Contains(norm, "english") || strings.Contains(norm, "fluency") || strings.Contains(norm, "speak") || strings.Contains(norm, "pronunciation") {
		// 4. Language learning intent
		reply = "Consistency is definitely the secret! Practicing out loud for even five minutes a day dramatically builds muscle memory. What part of speaking feels most challenging for you?"
		trans = "継続こそが最大の秘訣です！1日5分でも声に出して練習することで、口周りの筋肉の記憶が劇的に養われます。スピーキングのどの部分が一番難しく感じますか？"
		better = "「I want to be good at English」よりも「I'm aiming to speak more fluently and confidently」と表現すると具体的でスマートです。"
		tip = "「muscle memory」は [mʌsl-mɛməri] と発音し、'c' は発音しません（無音のc）。"
	} else if strings.HasSuffix(strings.TrimSpace(norm), "?") {
		// 5. User asked a general question
		reply = "That's an excellent question! In my view, it really depends on the situation and what you value most. What do you personally think is the most important factor?"
		trans = "素晴らしい質問ですね！私の考えでは、状況や何を一番重視するかによって大きく変わってきます。あなたご自身はどの要素が一番重要だと思いますか？"
		better = "質問された後に「That's a good question! Let me think...」と相槌を打つと、ネイティブらしい自然な会話の間が作れます。"
		tip = "「depends on」は [dɪ-pɛnd-zɑn] と完全に音が繋がってひとつの単語のようになります（リンキング）。"
	} else {
		// 6. General conversational progression
		switch level {
		case "beginner":
			reply = "I see! That sounds nice. Could you tell me a little bit more about that?"
			trans = "なるほど！良さそうですね。それについてもう少しだけ教えてもらえますか？"
			better = "相槌を打つときは「Yeah」だけでなく「I see!」「That makes sense!」を混ぜると表現力がアップします。"
			tip = "「a little bit」は [ə-lɪtl-bɪt] と発音し、アメリカ英語では 'tt' が軽い弾き音（ラ行に近い音）になります。"
		case "advanced":
			reply = "That's a very compelling point. It opens up quite a few interesting possibilities. Where do you see this heading in the near future?"
			trans = "とても説得力のある視点ですね。興味深い可能性がいくつも広がりそうです。近い将来、これはどう進展していくとお考えですか？"
			better = "「I think so too」の代わりに「I completely share that perspective」と言うと、プロフェッショナルな議論で一目置かれます。"
			tip = "「quite a few」は「かなり多くの」という意味のイディオムです。発音は滑らかに [kwaɪt-ə-fjuː] と繋ぎます。"
		default:
			reply = "That really makes sense. Thanks for sharing that perspective! What made you interested in this in the first place?"
			trans = "とても納得がいきます。その視点を共有してくれてありがとう！そもそも、それに興味を持ったきっかけは何だったのですか？"
			better = "自分の意見を述べるときは「To be honest...」や「From my perspective...」をクッションに置くと会話が滑らかになります。"
			tip = "「in the first place」は「そもそも・最初に」という定番表現です。リズムよくひと息で発声しましょう。"
		}
	}

	// Adapt for non-English languages
	if strings.HasPrefix(req.LangCode, "es") {
		reply = "¡Qué interesante! Me encanta tu punto de vista. ¿Podrías contarme un poco más sobre eso?"
		trans = "とても面白いですね！あなたの視点がとても気に入りました。それについてもう少し詳しく話してくれますか？"
	} else if strings.HasPrefix(req.LangCode, "fr") {
		reply = "C'est vraiment passionnant ! Je comprends tout à fait. Pourriez-vous m'en dire un peu plus à ce sujet ?"
		trans = "本当に魅力的ですね！とてもよく分かります。そのことについてもう少し詳しく話していただけますか？"
	} else if strings.HasPrefix(req.LangCode, "de") {
		reply = "Das ist ein sehr interessanter Gedanke! Könnten Sie mir bitte noch etwas mehr darüber berichten?"
		trans = "それはとても興味深い考えですね！それについてもう少し詳しく教えていただけますか？"
	} else if strings.HasPrefix(req.LangCode, "it") {
		reply = "Che bella idea! Mi piace molto il tuo punto di vista. Mi potresti raccontare qualcosa in più?"
		trans = "素敵なアイデアですね！あなたの視点はとても素晴らしいです。もう少し詳しく聞かせてもらえますか？"
	}

	sections := []struct {
		sec, text string
	}{
		{"reply", reply},
		{"translation", trans},
		{"better_phrasing", better},
		{"tip", tip},
	}

	for _, s := range sections {
		words := strings.Fields(s.text)
		for _, wStr := range words {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			g.sendDialogChunk(w, flusher, s.sec, wStr+" ")
			time.Sleep(20 * time.Millisecond)
		}
	}

	donePayload, _ := json.Marshal(model.DialogStreamChunk{Type: "done"})
	fmt.Fprintf(w, "data: %s\n\n", donePayload)
	flusher.Flush()
	return nil
}
