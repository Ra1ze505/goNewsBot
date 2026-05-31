package repository

//go:generate mockgen -source=ml.go -destination=../mocks/repository/ml_mock.go -package=mock_repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

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

const newsSummarySystemPrompt = `Ты редактор новостного дайджеста для Telegram.

Твоя задача — не пересказывать все сообщения подряд, а собрать короткую, полезную сводку дня.

Правила:
- Объединяй связанные сообщения в один топик.
- Удаляй рекламу, повторы, эмоциональные комментарии без фактов и малозначимые локальные детали.
- Выбирай только 3-5 самых важных топиков.
- Для каждого топика дай короткий заголовок и 1-2 предложения сути.
- Добавляй строку "Почему важно:" только если значение новости не очевидно.
- Не выдумывай факты, которых нет в сообщениях.
- Пиши на русском языке.
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
	systemPrompt  string
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
		systemPrompt:  newsSummarySystemPrompt,
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

	log.Infof("Creating summarization request for %d messages using Yandex AI Studio OpenAI-compatible API", len(messages))

	result, err := r.createChatCompletion(ctx, buildSummaryUserPrompt(messages))
	if err != nil {
		return "", err
	}

	log.Infof("Successfully received summarization result from Yandex AI Studio")

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

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type chatCompletionErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r *MLRepository) createChatCompletion(ctx context.Context, userPrompt string) (string, error) {
	token, err := r.tokenProvider.Token(ctx)
	if err != nil {
		return "", err
	}

	payload := chatCompletionRequest{
		Model: r.modelURI,
		Messages: []chatMessage{
			{Role: "system", Content: r.systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   r.maxTokens,
		Temperature: r.temperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chat completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-folder-id", r.folderID)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Yandex AI Studio chat completions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Yandex AI Studio returned %s: %s", resp.Status, parseChatCompletionError(resp.Body))
	}

	var completion chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", fmt.Errorf("failed to decode chat completion response: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("received empty chat completion response")
	}

	return completion.Choices[0].Message.Content, nil
}

func parseChatCompletionError(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "failed to read error response"
	}

	var errorResponse chatCompletionErrorResponse
	if err := json.Unmarshal(data, &errorResponse); err == nil && errorResponse.Error.Message != "" {
		return errorResponse.Error.Message
	}

	return strings.TrimSpace(string(data))
}

func buildSummaryUserPrompt(messages []string) string {
	var builder strings.Builder
	builder.WriteString("Сгруппируй эти сообщения канала в итоговый дайджест. Используй номера сообщений только для анализа, в финальный текст их не добавляй.\n\n")
	builder.WriteString("Сообщения:\n")
	for i, msg := range messages {
		builder.WriteString(fmt.Sprintf("\n[#%d]\n%s\n", i+1, strings.TrimSpace(msg)))
	}

	return builder.String()
}
