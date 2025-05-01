package repository

//go:generate mockgen -source=ml.go -destination=../mocks/repository/ml_mock.go -package=mock_repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	summaryPrompt = "Суммаризируй следующие новости и оставь только важное, не больше 2500 символов и 7 пунктов. " +
		"Не добавляй никаких комментариев. " +
		"Ответ должен быть на русском языке. " +
		"Убери любые призывы подписаться, подписаться на канал и т.д. " +
		"Это дожен быть обычный текст без символов: *`<>/ " +
		"Если среди новостей уже есть суммарная новость, то не добавляй её в список. " +
		"Пример:\n" +
		"🔸 Заголовок новости 1\n\n" +
		"🔸 Заголовок новости 2\n\n" +
		"🔸 Заголовок новости 3\n\n"
)

type MLRepositoryInterface interface {
	SummarizeMessages(messages []string) (string, error)
}

type MLRepository struct {
	apiToken string
	client   *http.Client
}

type OpenRouterRequest struct {
	Model     string              `json:"model"`
	Messages  []OpenRouterMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewMLRepository() (*MLRepository, error) {
	apiToken := os.Getenv("OPENROUTER_API_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("OPENROUTER_API_TOKEN environment variable must be set")
	}

	return &MLRepository{
		apiToken: apiToken,
		client:   &http.Client{Timeout: 250 * time.Second},
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
	// Combine all messages into one text
	combinedText := ""
	for _, msg := range messages {
		combinedText += msg + "\n\n"
	}

	reqBody := OpenRouterRequest{
		Model: "deepseek/deepseek-prover-v2:free",
		Messages: []OpenRouterMessage{
			{
				Role:    "system",
				Content: summaryPrompt,
			},
			{
				Role:    "user",
				Content: combinedText,
			},
		},
		MaxTokens: 500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiToken)

	log.Info("Sending request to OpenRouter")
	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := openRouterResp.Choices[0].Message.Content
	return cleanResponse(content), nil
}
