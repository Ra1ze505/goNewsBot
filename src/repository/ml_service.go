package repository

//go:generate mockgen -source=ml_service.go -destination=../mocks/repository/ml_service_mock.go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	summaryPrompt = "Суммаризируй следующие новости и оставь только важное. " +
		"Не добавляй никаких комментариев. " +
		"Ответ должен быть на русском языке. " +
		"Пункты отделяй друг от друга новой строкой и нумерацией. " +
		"Выводи в виде списка. " +
		"Можно использовать жирный шрифт для выделения важных моментов. " +
		"4000 символов максимум."
)

type MLServiceInterface interface {
	SummarizeMessages(messages []string) (string, error)
}

type MLService struct {
	apiToken string
	client   *http.Client
}

type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
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

func NewMLService() (*MLService, error) {
	apiToken := os.Getenv("OPENROUTER_API_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("OPENROUTER_API_TOKEN environment variable must be set")
	}

	return &MLService{
		apiToken: apiToken,
		client:   &http.Client{Timeout: 250 * time.Second},
	}, nil
}

func (s *MLService) SummarizeMessages(messages []string) (string, error) {
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
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	log.Info("Sending request to OpenRouter")
	resp, err := s.client.Do(req)
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

	return openRouterResp.Choices[0].Message.Content, nil
}
