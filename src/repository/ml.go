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

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
	log "github.com/sirupsen/logrus"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

const (
	defaultYandexOpenAIBaseURL = "https://llm.api.cloud.yandex.net/v1"
	defaultYandexModelName     = "yandexgpt/latest"
	defaultYandexMaxTokens     = 1800
	defaultYandexTemperature   = 0.2
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
	SummarizeMessages(messages []string) (string, error)
}

type MLRepository struct {
	httpClient    *http.Client
	tokenProvider tokenProvider
	baseURL       string
	folderID      string
	modelURI      string
	extractPrompt string
	renderPrompt  string
	maxTokens     int
	temperature   float64
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

	baseURL := strings.TrimRight(os.Getenv("YANDEX_OPENAI_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultYandexOpenAIBaseURL
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
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		maxTokens:     defaultYandexMaxTokens,
		temperature:   defaultYandexTemperature,
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

func (r *MLRepository) createChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	token, err := r.tokenProvider.Token(ctx)
	if err != nil {
		return "", err
	}

	client := openai.NewClient(
		option.WithAPIKey(token),
		option.WithBaseURL(r.baseURL),
		option.WithHeader("x-folder-id", r.folderID),
		option.WithHTTPClient(r.httpClient),
		option.WithRequestTimeout(yandexRequestTimeout),
	)

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(r.modelURI),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		MaxTokens:   param.NewOpt(int64(r.maxTokens)),
		Temperature: param.NewOpt(r.temperature),
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
	rawPlan, err := r.createChatCompletion(ctx, r.extractPrompt, buildTopicExtractionPrompt(messages))
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

	return r.createChatCompletion(ctx, r.renderPrompt, prompt)
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
