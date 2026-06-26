package repository

//go:generate mockgen -source=ml.go -destination=../mocks/repository/ml_mock.go -package=mock_repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
	log "github.com/sirupsen/logrus"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

const (
	defaultYandexOpenAIBaseURL = "https://llm.api.cloud.yandex.net/v1/"
	defaultYandexModelName     = "yandexgpt/latest"
	defaultExtractMaxTokens    = 16000
	defaultExtractTemperature  = 0.2
	defaultRenderMaxTokens     = 8000
	defaultRenderTemperature   = 1.0
	yandexRequestTimeout       = 300 * time.Second
)

const topicExtractionSystemPrompt = `Ты аналитик новостной редакции.

Твоя задача — превратить поток сообщений Telegram-канала в структурированный план дайджеста.

Правила:
- Объединяй связанные сообщения в один топик, даже если они написаны разными словами.
- Удаляй рекламу, повторы, эмоциональные комментарии без фактов, анонсы без сути и малозначимые локальные детали.
- Выбирай не больше 5 топиков.
- Оцени важность от 1 до 5.
- Сохраняй номера исходных сообщений, из которых собран топик.
- Не выдумывай факты, которых нет в сообщениях.
- Верни только валидный JSON без markdown.`

const topicsExtractionSystemPrompt = `Ты аналитик новостной редакции.

Преврати поток сообщений Telegram-канала в список топиков.

Правила:
- Объединяй связанные сообщения в один топик, даже если они написаны разными словами.
- Удаляй рекламу, повторы, эмоциональные комментарии без фактов.
- Не больше 8 топиков.
- Оцени важность от 1 до 5.
- Укажи рубрику (одно из: военное, происшествия, экономика, политика, общество, другое).
- Сохраняй номера исходных сообщений, из которых собран топик.
- Не выдумывай факты, которых нет в сообщениях.
- Верни только валидный JSON без markdown.
Схема: {"topics":[{"title","summary","category","importance","source_message_numbers":[..]}]}`

const matchConfirmSystemPrompt = `Ты аналитик, который отслеживает развитие сюжетов в новостном канале.

Тебе дан сегодняшний топик и несколько похожих существующих сюжетов канала.
Реши: топик — это продолжение одного из сюжетов или это НОВЫЙ сюжет?
Не объединяй разные по сути сюжеты.
Верни только валидный JSON без markdown: {"matched_id": <id или null>, "is_new": <true|false>, "reason": "коротко"}.`

const deltaSystemPrompt = `Ты аналитик, который ведёт хронику сюжета в новостном канале.

Дан сюжет (его текущее состояние), сегодняшние сообщения по нему и статистика-факты.
Статистику не оспаривай и не выдумывай иное.
Задача:
1) "delta_summary" — кратко что нового именно сегодня (1-2 предложения; может быть пустым, если новизны нет);
2) "state" — обновлённое состояние сюжета, до 600 символов, без воды и без номеров сообщений.
Верни только валидный JSON без markdown: {"delta_summary":"...","state":"..."}.`

const groupedDigestSystemPrompt = `Ты выпускающий редактор Telegram-дайджеста.

Собери дайджест по сгруппированным сюжетам. Группы и порядок (заголовок группы выводи только если в ней есть сюжеты):
🆕 Новое — сюжеты из группы new;
🔺 Эскалация — сюжеты из группы escalation;
▶️ Развитие — сюжеты из группы ongoing.
Для каждого сюжета: жирный заголовок и 1-2 предложения сути из его delta_summary/state.
Группу recurring_noise НЕ перечисляй по одному — сверни в одну строку в конце:
"Фон без изменений: <рубрики/темы через запятую>".
Пиши по-русски, только факты из входных данных, без номеров сообщений, итог < 3500 символов.
Верни только готовый текст без markdown-ограждений.`

const digestRenderSystemPrompt = `Ты выпускающий редактор Telegram-дайджеста.

Твоя задача — написать финальную сводку по готовому JSON-плану топиков.

Правила:
- Пиши на русском языке.
- Не добавляй факты, которых нет в JSON.
- Сохраняй только 3-5 самых важных топиков.
- Формат каждого топика: жирный номер и заголовок, затем 1-2 предложения сути.
- Добавляй "Почему важно:" только если это реально помогает понять значение новости.
- Не показывай номера исходных сообщений.
- Итог должен быть короче 3500 символов.`

type MLRepositoryInterface interface {
	// стадия A: извлечение топиков с рубриками и номерами исходных сообщений
	ExtractTopics(messages []MessageInput) ([]CandidateTopic, error)
	// стадия B: эмбеддинги (text-search-doc / text-search-query, 256d)
	EmbedDocuments(texts []string) ([][]float32, error)
	EmbedQueries(texts []string) ([][]float32, error)
	// стадия C: подтверждение матчинга в "серой зоне"
	ConfirmMatch(cand CandidateTopic, options []StorylineBrief) (matchedID int64, isNew bool, err error)
	// стадия D: дельта + обновлённое состояние сюжета
	WriteDelta(in DeltaInput) (newState string, deltaSummary string, err error)
	// стадия F: рендер сгруппированного дайджеста
	RenderDigest(groups DigestGroups) (string, error)

	// обратная совместимость на время миграции (использует scripts/historical_summary)
	SummarizeMessages(messages []string) (string, error)
}

// CandidateTopic - топик дня, извлечённый из сообщений (стадия A).
type CandidateTopic struct {
	Title                string
	Summary              string
	Category             string
	Importance           int
	SourceMessageNumbers []int   // позиционные номера в дневной пачке (1-based)
	SourceMessageIDs     []int64 // реальные message_id, резолвятся в сервисе
}

// StorylineBrief - краткая карточка сюжета для LLM-подтверждения матчинга (стадия C).
type StorylineBrief struct {
	ID         int64
	Title      string
	State      string
	LastSeen   string
	AvgCount   float64
	Similarity float64
}

// DeltaInput - вход для расчёта дельты и обновления состояния сюжета (стадия D).
type DeltaInput struct {
	Title            string
	CurrentState     string
	TodayCount       int
	DaysSeen         int
	WindowDays       int
	MedianCount      float64
	MedianImportance float64
	ChangeType       string
	TodayMessages    []string
}

// DigestItem - один сюжет в сгруппированном дайджесте (стадия F).
type DigestItem struct {
	Title        string
	DeltaSummary string
	State        string
	Category     string
	Importance   int
}

// DigestGroups - сгруппированные сюжеты для финального рендера (стадия F).
type DigestGroups struct {
	New            []DigestItem
	Escalation     []DigestItem
	Ongoing        []DigestItem // ongoing/deescalation с непустым delta_summary
	RecurringNoise []string     // рубрики/темы фона
}

type chatCompletionParams struct {
	MaxTokens   int
	Temperature float64
}

type MLRepository struct {
	httpClient    *http.Client
	tokenProvider tokenProvider
	baseURL       string
	folderID      string
	modelURI      string
	embedDocURI   string
	embedQueryURI string
	extractPrompt string
	renderPrompt  string
	extractParams chatCompletionParams
	renderParams  chatCompletionParams
}

func NewMLRepository() (*MLRepository, error) {
	ctx := context.Background()

	folderID := os.Getenv("YANDEX_FOLDER_ID")
	if folderID == "" {
		return nil, fmt.Errorf("YANDEX_FOLDER_ID environment variable is required")
	}

	tokenProvider, err := newYandexTokenProvider(ctx)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimSpace(os.Getenv("YANDEX_OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultYandexOpenAIBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	modelURI := os.Getenv("YANDEX_MODEL_URI")
	if modelURI == "" {
		modelURI = fmt.Sprintf("gpt://%s/%s", folderID, defaultYandexModelName)
	}

	return &MLRepository{
		httpClient:    &http.Client{Timeout: yandexRequestTimeout},
		tokenProvider: tokenProvider,
		baseURL:       baseURL,
		folderID:      folderID,
		modelURI:      modelURI,
		embedDocURI:   config.EmbedDocURI(folderID),
		embedQueryURI: config.EmbedQueryURI(folderID),
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		extractParams: chatCompletionParams{
			MaxTokens:   defaultExtractMaxTokens,
			Temperature: defaultExtractTemperature,
		},
		renderParams: chatCompletionParams{
			MaxTokens:   defaultRenderMaxTokens,
			Temperature: defaultRenderTemperature,
		},
	}, nil
}

func cleanResponse(content string) string {
	content = regexp.MustCompile("```[a-zA-Z]*\n").ReplaceAllString(content, "")
	content = regexp.MustCompile("```").ReplaceAllString(content, "")
	content = regexp.MustCompile("---\n").ReplaceAllString(content, "")
	content = strings.TrimSpace(content)
	return content
}

func (r *MLRepository) SummarizeMessages(messages []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), yandexRequestTimeout)
	defer cancel()

	log.Infof("Creating topic extraction request for %d messages using Yandex AI Studio OpenAI-compatible API", len(messages))

	topicPlan, err := r.extractTopicPlan(ctx, messages)
	if err != nil {
		return "", err
	}
	if len(topicPlan.Topics) == 0 {
		log.Infof("No significant topics found in %d messages", len(messages))
		return "За последние сутки значимых новостей не найдено.", nil
	}

	log.Infof("Rendering digest from %d extracted topics", len(topicPlan.Topics))

	result, err := r.renderDigest(ctx, topicPlan)
	if err != nil {
		return "", err
	}

	log.Infof("Successfully received digest from Yandex AI Studio")

	return cleanResponse(result), nil
}

type tokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type staticTokenProvider struct {
	token string
}

func (p staticTokenProvider) Token(context.Context) (string, error) {
	return p.token, nil
}

type iamTokenProvider struct {
	sdk *ycsdk.SDK
}

func (p *iamTokenProvider) Token(ctx context.Context) (string, error) {
	token, err := p.sdk.CreateIAMToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get IAM token: %w", err)
	}
	return token.IamToken, nil
}

func newYandexTokenProvider(ctx context.Context) (tokenProvider, error) {
	apiKey := os.Getenv("YANDEX_API_KEY")
	if apiKey != "" {
		return staticTokenProvider{token: apiKey}, nil
	}

	serviceAccountKeyPath := os.Getenv("YANDEX_SERVICE_ACCOUNT_KEY_PATH")
	if serviceAccountKeyPath == "" {
		return nil, fmt.Errorf("YANDEX_API_KEY or YANDEX_SERVICE_ACCOUNT_KEY_PATH environment variable is required")
	}

	keyData, err := os.ReadFile(serviceAccountKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account key: %w", err)
	}

	key, err := iamkey.ReadFromJSONBytes(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account key: %w", err)
	}

	creds, err := ycsdk.ServiceAccountKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: creds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex Cloud SDK: %w", err)
	}

	return &iamTokenProvider{sdk: sdk}, nil
}

type summaryTopicPlan struct {
	Topics              []summaryTopic `json:"topics"`
	NoiseMessageNumbers []int          `json:"noise_message_numbers,omitempty"`
}

type summaryTopic struct {
	Title                string `json:"title"`
	Summary              string `json:"summary"`
	WhyImportant         string `json:"why_important,omitempty"`
	Importance           int    `json:"importance"`
	SourceMessageNumbers []int  `json:"source_message_numbers"`
}

func (r *MLRepository) newClient(token string) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(token),
		option.WithBaseURL(r.baseURL),
		option.WithHeader("x-folder-id", r.folderID),
		option.WithHTTPClient(r.httpClient),
		option.WithRequestTimeout(yandexRequestTimeout),
	)
}

func (r *MLRepository) createChatCompletion(ctx context.Context, params chatCompletionParams, systemPrompt, userPrompt string) (string, error) {
	token, err := r.tokenProvider.Token(ctx)
	if err != nil {
		return "", err
	}

	client := r.newClient(token)

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(r.modelURI),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		MaxTokens:   param.NewOpt(int64(params.MaxTokens)),
		Temperature: param.NewOpt(params.Temperature),
	})
	if err != nil {
		return "", fmt.Errorf("failed to call Yandex AI Studio chat completions: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("received empty chat completion response")
	}

	return completion.Choices[0].Message.Content, nil
}

func (r *MLRepository) extractTopicPlan(ctx context.Context, messages []string) (*summaryTopicPlan, error) {
	rawPlan, err := r.createChatCompletion(ctx, r.extractParams, r.extractPrompt, buildTopicExtractionPrompt(messages))
	if err != nil {
		return nil, fmt.Errorf("failed to extract news topics: %w", err)
	}

	topicPlan, err := parseSummaryTopicPlan(rawPlan, len(messages))
	if err != nil {
		return nil, fmt.Errorf("failed to parse news topic plan: %w", err)
	}

	return topicPlan, nil
}

func (r *MLRepository) renderDigest(ctx context.Context, topicPlan *summaryTopicPlan) (string, error) {
	prompt, err := buildDigestRenderPrompt(topicPlan)
	if err != nil {
		return "", err
	}

	return r.createChatCompletion(ctx, r.renderParams, r.renderPrompt, prompt)
}

func buildTopicExtractionPrompt(messages []string) string {
	var builder strings.Builder
	builder.WriteString("Проанализируй сообщения канала и верни JSON строго по схеме:\n")
	builder.WriteString(`{"topics":[{"title":"короткий заголовок","summary":"фактическая суть топика","why_important":"почему это важно, если нужно","importance":5,"source_message_numbers":[1,2]}],"noise_message_numbers":[3]}`)
	builder.WriteString("\n\n")
	builder.WriteString("Если значимых топиков нет, верни {\"topics\":[],\"noise_message_numbers\":[...]}.\n")
	builder.WriteString("Используй номера сообщений только как ссылки на источники.\n\n")
	builder.WriteString("Сообщения:\n")
	for i, msg := range messages {
		builder.WriteString(fmt.Sprintf("\n[#%d]\n%s\n", i+1, strings.TrimSpace(msg)))
	}

	return builder.String()
}

func buildDigestRenderPrompt(topicPlan *summaryTopicPlan) (string, error) {
	data, err := json.MarshalIndent(topicPlan, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal topic plan: %w", err)
	}

	return "Напиши финальный Telegram-дайджест по этому JSON-плану:\n\n" + string(data), nil
}

func parseSummaryTopicPlan(content string, messageCount int) (*summaryTopicPlan, error) {
	var topicPlan summaryTopicPlan
	if err := json.Unmarshal([]byte(cleanResponse(content)), &topicPlan); err != nil {
		return nil, err
	}

	normalizedTopics := make([]summaryTopic, 0, len(topicPlan.Topics))
	for _, topic := range topicPlan.Topics {
		topic.Title = strings.TrimSpace(topic.Title)
		topic.Summary = strings.TrimSpace(topic.Summary)
		topic.WhyImportant = strings.TrimSpace(topic.WhyImportant)
		if topic.Title == "" || topic.Summary == "" {
			continue
		}
		if topic.Importance < 1 {
			topic.Importance = 1
		}
		if topic.Importance > 5 {
			topic.Importance = 5
		}
		topic.SourceMessageNumbers = validMessageNumbers(topic.SourceMessageNumbers, messageCount)
		normalizedTopics = append(normalizedTopics, topic)
	}

	sort.SliceStable(normalizedTopics, func(i, j int) bool {
		return normalizedTopics[i].Importance > normalizedTopics[j].Importance
	})
	if len(normalizedTopics) > 5 {
		normalizedTopics = normalizedTopics[:5]
	}

	topicPlan.Topics = normalizedTopics
	topicPlan.NoiseMessageNumbers = validMessageNumbers(topicPlan.NoiseMessageNumbers, messageCount)

	return &topicPlan, nil
}

func validMessageNumbers(numbers []int, messageCount int) []int {
	validNumbers := make([]int, 0, len(numbers))
	seen := make(map[int]struct{}, len(numbers))
	for _, number := range numbers {
		if number < 1 || number > messageCount {
			continue
		}
		if _, ok := seen[number]; ok {
			continue
		}
		seen[number] = struct{}{}
		validNumbers = append(validNumbers, number)
	}
	return validNumbers
}

// --- Storyline Tracking / TDT ---

type candidateTopicsPlan struct {
	Topics []candidateTopicJSON `json:"topics"`
}

type candidateTopicJSON struct {
	Title                string `json:"title"`
	Summary              string `json:"summary"`
	Category             string `json:"category"`
	Importance           int    `json:"importance"`
	SourceMessageNumbers []int  `json:"source_message_numbers"`
}

// ExtractTopics - стадия A: извлекает топики дня с рубриками и номерами сообщений.
func (r *MLRepository) ExtractTopics(messages []MessageInput) ([]CandidateTopic, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), yandexRequestTimeout)
	defer cancel()

	texts := make([]string, len(messages))
	for i, m := range messages {
		texts[i] = m.Text
	}

	raw, err := r.createChatCompletion(ctx, r.extractParams, topicsExtractionSystemPrompt, buildTopicsExtractionPrompt(texts))
	if err != nil {
		return nil, fmt.Errorf("failed to extract topics: %w", err)
	}

	var plan candidateTopicsPlan
	if err := json.Unmarshal([]byte(cleanResponse(raw)), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse topics plan: %w", err)
	}

	candidates := make([]CandidateTopic, 0, len(plan.Topics))
	for _, t := range plan.Topics {
		title := strings.TrimSpace(t.Title)
		summary := strings.TrimSpace(t.Summary)
		if title == "" || summary == "" {
			continue
		}
		importance := t.Importance
		if importance < 1 {
			importance = 1
		}
		if importance > 5 {
			importance = 5
		}
		candidates = append(candidates, CandidateTopic{
			Title:                title,
			Summary:              summary,
			Category:             strings.TrimSpace(t.Category),
			Importance:           importance,
			SourceMessageNumbers: validMessageNumbers(t.SourceMessageNumbers, len(messages)),
		})
	}
	return candidates, nil
}

func buildTopicsExtractionPrompt(messages []string) string {
	var b strings.Builder
	b.WriteString("Проанализируй сообщения канала и верни JSON строго по схеме:\n")
	b.WriteString(`{"topics":[{"title":"короткий заголовок","summary":"фактическая суть","category":"экономика","importance":5,"source_message_numbers":[1,2]}]}`)
	b.WriteString("\n\nЕсли значимых топиков нет, верни {\"topics\":[]}.\n")
	b.WriteString("Сообщения:\n")
	for i, msg := range messages {
		b.WriteString(fmt.Sprintf("\n[#%d]\n%s\n", i+1, strings.TrimSpace(msg)))
	}
	return b.String()
}

// EmbedDocuments - эмбеддинги документов (состояний сюжетов) моделью text-search-doc.
func (r *MLRepository) EmbedDocuments(texts []string) ([][]float32, error) {
	return r.embed(r.embedDocURI, texts)
}

// EmbedQueries - эмбеддинги запросов (топиков-кандидатов) моделью text-search-query.
func (r *MLRepository) EmbedQueries(texts []string) ([][]float32, error) {
	return r.embed(r.embedQueryURI, texts)
}

func (r *MLRepository) embed(modelURI string, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), yandexRequestTimeout)
	defer cancel()

	token, err := r.tokenProvider.Token(ctx)
	if err != nil {
		return nil, err
	}
	client := r.newClient(token)

	// Yandex эмбеддинги принимают по одному тексту на запрос — эмбеддим поштучно.
	result := make([][]float32, 0, len(texts))
	for _, text := range texts {
		resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Model: openai.EmbeddingModel(modelURI),
			Input: openai.EmbeddingNewParamsInputUnion{OfString: param.NewOpt(text)},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding: %w", err)
		}
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("received empty embedding response")
		}
		result = append(result, float64sToFloat32s(resp.Data[0].Embedding))
	}
	return result, nil
}

func float64sToFloat32s(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}

type confirmMatchResponse struct {
	MatchedID *int64 `json:"matched_id"`
	IsNew     bool   `json:"is_new"`
	Reason    string `json:"reason"`
}

// ConfirmMatch - стадия C: LLM решает, продолжение это одного из сюжетов или новый сюжет.
func (r *MLRepository) ConfirmMatch(cand CandidateTopic, options []StorylineBrief) (int64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), yandexRequestTimeout)
	defer cancel()

	raw, err := r.createChatCompletion(ctx, r.extractParams, matchConfirmSystemPrompt, buildMatchConfirmPrompt(cand, options))
	if err != nil {
		return 0, false, fmt.Errorf("failed to confirm match: %w", err)
	}

	var resp confirmMatchResponse
	if err := json.Unmarshal([]byte(cleanResponse(raw)), &resp); err != nil {
		return 0, false, fmt.Errorf("failed to parse match confirmation: %w", err)
	}

	if resp.IsNew || resp.MatchedID == nil {
		return 0, true, nil
	}
	// Защита: id должен быть среди предложенных вариантов.
	for _, o := range options {
		if o.ID == *resp.MatchedID {
			return *resp.MatchedID, false, nil
		}
	}
	return 0, true, nil
}

func buildMatchConfirmPrompt(cand CandidateTopic, options []StorylineBrief) string {
	type optionJSON struct {
		ID         int64   `json:"id"`
		Title      string  `json:"title"`
		State      string  `json:"state"`
		LastSeen   string  `json:"last_seen"`
		AvgCount   float64 `json:"avg_count"`
		Similarity float64 `json:"similarity"`
	}
	opts := make([]optionJSON, len(options))
	for i, o := range options {
		opts[i] = optionJSON{
			ID:         o.ID,
			Title:      o.Title,
			State:      o.State,
			LastSeen:   o.LastSeen,
			AvgCount:   o.AvgCount,
			Similarity: o.Similarity,
		}
	}
	payload := map[string]any{
		"topic":      map[string]any{"title": cand.Title, "summary": cand.Summary, "category": cand.Category},
		"storylines": opts,
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	return "Данные:\n" + string(data)
}

type writeDeltaResponse struct {
	DeltaSummary string `json:"delta_summary"`
	State        string `json:"state"`
}

// WriteDelta - стадия D: дельта дня и обновлённое состояние сюжета.
func (r *MLRepository) WriteDelta(in DeltaInput) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), yandexRequestTimeout)
	defer cancel()

	raw, err := r.createChatCompletion(ctx, r.renderParams, deltaSystemPrompt, buildDeltaPrompt(in))
	if err != nil {
		return "", "", fmt.Errorf("failed to write delta: %w", err)
	}

	var resp writeDeltaResponse
	if err := json.Unmarshal([]byte(cleanResponse(raw)), &resp); err != nil {
		return "", "", fmt.Errorf("failed to parse delta response: %w", err)
	}

	state := strings.TrimSpace(resp.State)
	if state == "" {
		state = strings.TrimSpace(in.CurrentState)
	}
	return state, strings.TrimSpace(resp.DeltaSummary), nil
}

func buildDeltaPrompt(in DeltaInput) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Сюжет: %s\n", in.Title))
	b.WriteString(fmt.Sprintf("Текущее состояние: %s\n\n", in.CurrentState))
	b.WriteString("Статистика (факты):\n")
	b.WriteString(fmt.Sprintf("- встречался %d из %d дней окна\n", in.DaysSeen, in.WindowDays))
	b.WriteString(fmt.Sprintf("- медиана %.1f сообщений/день, сегодня %d\n", in.MedianCount, in.TodayCount))
	b.WriteString(fmt.Sprintf("- медианная важность %.1f\n", in.MedianImportance))
	b.WriteString(fmt.Sprintf("- предварительная метка изменения: %s\n\n", in.ChangeType))
	b.WriteString("Сегодняшние сообщения по сюжету:\n")
	for i, msg := range in.TodayMessages {
		b.WriteString(fmt.Sprintf("\n[#%d]\n%s\n", i+1, strings.TrimSpace(msg)))
	}
	return b.String()
}

// RenderDigest - стадия F: финальный сгруппированный дайджест.
func (r *MLRepository) RenderDigest(groups DigestGroups) (string, error) {
	if len(groups.New) == 0 && len(groups.Escalation) == 0 && len(groups.Ongoing) == 0 && len(groups.RecurringNoise) == 0 {
		return "За последние сутки значимых новостей не найдено.", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), yandexRequestTimeout)
	defer cancel()

	prompt, err := buildGroupedDigestPrompt(groups)
	if err != nil {
		return "", err
	}

	result, err := r.createChatCompletion(ctx, r.renderParams, groupedDigestSystemPrompt, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to render digest: %w", err)
	}
	return cleanResponse(result), nil
}

func buildGroupedDigestPrompt(groups DigestGroups) (string, error) {
	data, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal digest groups: %w", err)
	}
	return "Собери Telegram-дайджест по этим сгруппированным сюжетам:\n\n" + string(data), nil
}
