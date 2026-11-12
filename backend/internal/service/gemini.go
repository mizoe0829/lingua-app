package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

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
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	return &GeminiService{
		apiKey: apiKey,
		model:  modelName,
		client: &http.Client{Timeout: 60 * time.Second},
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
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return g.streamFallbackDialog(ctx, req, flusher, w)
	}
	defer resp.Body.Close()

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

func (g *GeminiService) streamFallbackDialog(ctx context.Context, req model.DialogStreamRequest, flusher http.Flusher, w http.ResponseWriter) error {
	reply := fmt.Sprintf("That sounds really interesting! Could you tell me more about what you like about it?")
	trans := fmt.Sprintf("それはとても興味深いですね！それのどんなところが好きなのか、もっと詳しく教えてもらえますか？")
	better := fmt.Sprintf("「%s」は通じますが、ネイティブなら「I'd like to talk about...」や「In my experience...」と切り出すとより自然です。", req.UserMessage)
	tip := fmt.Sprintf("文頭の単語にピッチの山を置き、文末に向かって少し息を抜きながら音を下げると英語らしいイントネーションになります。")

	if strings.HasPrefix(req.LangCode, "es") {
		reply = "¡Qué interesante! ¿Podrías contarme un poco más sobre eso?"
		trans = "とても面白いですね！それについてもう少し話してくれますか？"
	} else if strings.HasPrefix(req.LangCode, "fr") {
		reply = "C'est très intéressant ! Pourriez-vous m'en dire plus à ce sujet ?"
		trans = "それはとても興味深いですね！そのことについてもっと話していただけますか？"
	} else if strings.HasPrefix(req.LangCode, "de") {
		reply = "Das klingt sehr interessant! Könnten Sie mir mehr darüber erzählen?"
		trans = "それはとても面白そうですね！それについてもっと詳しく教えていただけますか？"
	} else if strings.HasPrefix(req.LangCode, "it") {
		reply = "Molto interessante! Mi potresti dire qualcosa in più al riguardo?"
		trans = "とても面白いですね！それについてもう少し教えてもらえますか？"
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
			time.Sleep(25 * time.Millisecond)
		}
	}

	donePayload, _ := json.Marshal(model.DialogStreamChunk{Type: "done"})
	fmt.Fprintf(w, "data: %s\n\n", donePayload)
	flusher.Flush()
	return nil
}
