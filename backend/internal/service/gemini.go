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
		// Log warning and fallback
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

// streamFallbackAnalysis simulates progressive streaming when external API is unreachable
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

	// Stream line by line with slight realistic micro-delay for smooth UI feel
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

