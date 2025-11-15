package repository

//go:generate mockgen -source=ml.go -destination=../mocks/repository/ml_mock.go -package=mock_repository

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// Batch size for iterative summarization
	batchSize = 10
)

type MLRepositoryInterface interface {
	SummarizeMessages(messages []string) (string, error)
}

type MLRepository struct {
	yandexClient  *YandexClient
	threadManager *ThreadManager
	assistantID   string
}

func NewMLRepository() (*MLRepository, error) {
	ctx := context.Background()

	// Получаем конфигурацию из переменных окружения
	serviceAccountKeyPath := os.Getenv("YANDEX_SERVICE_ACCOUNT_KEY_PATH")
	if serviceAccountKeyPath == "" {
		return nil, fmt.Errorf("YANDEX_SERVICE_ACCOUNT_KEY_PATH environment variable is required")
	}

	folderID := os.Getenv("YANDEX_FOLDER_ID")
	if folderID == "" {
		return nil, fmt.Errorf("YANDEX_FOLDER_ID environment variable is required")
	}

	assistantID := os.Getenv("YANDEX_ASSISTANT_ID")
	if assistantID == "" {
		return nil, fmt.Errorf("YANDEX_ASSISTANT_ID environment variable is required")
	}

	// Создаем Yandex Cloud клиент
	yandexClient, err := NewYandexClient(ctx, serviceAccountKeyPath, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex Cloud client: %w", err)
	}

	// Создаем менеджер тредов
	threadManager, err := NewThreadManager(yandexClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create thread manager: %w", err)
	}

	return &MLRepository{
		yandexClient:  yandexClient,
		threadManager: threadManager,
		assistantID:   assistantID,
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
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	log.Infof("Creating summarization request for %d messages using Yandex Cloud AI", len(messages))

	threadID, err := r.threadManager.CreateThread(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create thread: %w", err)
	}

	log.Debugf("Created thread with ID: %s", threadID)

	// Split messages into batches
	totalBatches := (len(messages) + batchSize - 1) / batchSize
	log.Infof("Processing %d messages in %d batches (batch size: %d)", len(messages), totalBatches, batchSize)

	// Process each batch
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]
		batchNum := (i / batchSize) + 1

		// Combine messages in the batch
		combinedText := ""
		for _, msg := range batch {
			combinedText += msg + "\n\n"
		}

		log.Debugf("Adding batch %d/%d (%d messages) to thread %s", batchNum, totalBatches, len(batch), threadID)

		err = r.threadManager.AddMessage(ctx, threadID, combinedText)
		if err != nil {
			return "", fmt.Errorf("failed to add batch %d to thread: %w", batchNum, err)
		}

		log.Debugf("Successfully added batch %d/%d to thread %s", batchNum, totalBatches, threadID)
	}

	log.Infof("All %d batches added to thread %s, running assistant", totalBatches, threadID)

	result, err := r.threadManager.RunAssistant(ctx, threadID, r.assistantID)
	if err != nil {
		return "", fmt.Errorf("failed to run assistant: %w", err)
	}

	log.Infof("Successfully received summarization result from Yandex Cloud AI")

	cleanedResult := cleanResponse(result)

	return cleanedResult, nil
}
