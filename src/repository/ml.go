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
	summarySystemPrompt = `
  Ты — ИИ-ассистент по созданию сводок новостей. Строго соблюдай правила:
  1. Излагай только факты, без пояснений или вступлений.
  2. Каждый пункт начинается с 🔸. Только этот символ.
  3. В каждом пункте максимум 2 коротких предложения. Пункты цельные, не разбивай и не продолжай их вне строки.
  4. Между пунктами — одна пустая строка.
  5. Только русский язык.
  `

	summaryUserPrompt = `
  Выдели 5-10 главных событий из новостей ниже (объединяй схожие). Если новостей мало — добавь важные факты из общего контекста до 5 пунктов.
  
  Формат:
  
  🔸 Одно событие
  
  🔸 Второе событие
  
  Новости:
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
		MaxTokens:   1000,
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

	log.Infof("response from openrouter.ai: %s", string(body))

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
