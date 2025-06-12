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
)

const (
	summarySystemPrompt = `
   Ты — ИИ-ассистент для структурирования новостей. Твоя задача: создавать лаконичные сводки, строго соблюдая правила:
   1. Выводи ТОЛЬКО содержание новостей без пояснений
   2. Формат: пункты с 🔸, разделённые пустой строкой
   3. Никогда не прерывай пункт на середине
   4. Не используй другие символы, кроме 🔸.
`
	summaryUserPrompt = `
   Суммаризируй следующие новости, соблюдая правила:
   - Выбери 5-10 ГЛАВНЫХ событий (объединяя связанные)
   - Каждый пункт: максимум 2 предложения, начинается с 🔸
   - Только русский язык
   - Строго 5-10 пунктов (если новостей мало - дополни до 7 важными фактами из контекста)
   - Объём всего ответа: 1200-2500 символов
   - Никаких других символов/форматирования кроме 🔸 и пробелов
   - Между пунктами ВСЕГДА пустая строка
   
   Пример формата:
   🔸 Пункт 1
   
   🔸 Пункт 2
   
   Новости для обработки:
`
)

type MLRepositoryInterface interface {
	SummarizeMessages(messages []string) (string, error)
}

type MLRepository struct {
	apiToken string
	client   *http.Client
}

type OpenRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenRouterMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
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
		Model: "deepseek/deepseek-r1-0528-qwen3-8b:free",
		Messages: []OpenRouterMessage{
			{
				Role:    "system",
				Content: summarySystemPrompt,
			},
			{
				Role:    "user",
				Content: summaryUserPrompt + "\n" + combinedText,
			},
		},
		MaxTokens:   800,
		Temperature: 0.3,
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
	if content == "" {
		return "", fmt.Errorf("no content in response")
	}

	return cleanResponse(content), nil
}
