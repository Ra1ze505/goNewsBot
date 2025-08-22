package repository

//go:generate mockgen -source=ml.go -destination=../mocks/repository/ml_mock.go -package=mock_repository

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	summaryPrompt = `Ты — ИИ-ассистент по созданию сводок новостей. СТРОГО соблюдай ВСЕ правила:

ФОРМАТИРОВАНИЕ:
1. Каждый пункт начинается ТОЛЬКО с символа 🔸 и пробела
2. Каждый пункт — это ОДНА строка, НЕ разрывай пункты переносами
3. Между пунктами — ОДНА пустая строка
4. НЕ используй дополнительные символы, только 🔸

СОДЕРЖАНИЕ:
5. В каждом пункте МАКСИМУМ 25 слов
6. Максимум 2 коротких простых предложения в пункте
7. Излагай только факты, без комментариев
8. Объединяй схожие события в один пункт
9. Только русский язык

ПРОВЕРКА:
10. Каждый пункт должен быть короче 120 символов
11. Если пункт длиннее — сократи его
12. Всегда давай ответ, даже если новостей мало

Пример правильного формата:
🔸 Короткое событие. Второй факт.

🔸 Другое событие в одной строке.

Создай сводку из 5-8 главных событий. Каждый пункт ОБЯЗАТЕЛЬНО в одной строке, максимум 25 слов.

Новости:`
)

type MLRepositoryInterface interface {
	SummarizeMessages(messages []string) (string, error)
}

type MLRepository struct {
	client *http.Client
}

type CreateJobResponse struct {
	Code   int `json:"code"`
	Result struct {
		JobID    string `json:"job_id"`
		Language string `json:"language"`
	} `json:"result"`
	Message struct {
		En string `json:"en"`
		Zh string `json:"zh"`
	} `json:"message"`
}

type SSEData struct {
	State int    `json:"state"`
	Data  string `json:"data"`
}

func NewMLRepository() (*MLRepository, error) {
	return &MLRepository{
		client: &http.Client{Timeout: 300 * time.Second},
	}, nil
}

func cleanResponse(content string) string {
	content = regexp.MustCompile("```[a-zA-Z]*\n").ReplaceAllString(content, "")
	content = regexp.MustCompile("```").ReplaceAllString(content, "")
	content = regexp.MustCompile("---\n").ReplaceAllString(content, "")
	content = strings.TrimSpace(content)
	return content
}

func (r *MLRepository) createJob(text string) (string, error) {
	log.Debugf("Creating job with text length: %d", len(text))

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	if err := w.WriteField("mode", "Summary"); err != nil {
		return "", fmt.Errorf("error writing mode field: %w", err)
	}
	if err := w.WriteField("file_type", "text"); err != nil {
		return "", fmt.Errorf("error writing file_type field: %w", err)
	}
	if err := w.WriteField("entertext", text); err != nil {
		return "", fmt.Errorf("error writing entertext field: %w", err)
	}
	if err := w.WriteField("language", "Русский"); err != nil {
		return "", fmt.Errorf("error writing language field: %w", err)
	}
	if err := w.WriteField("length", "medium"); err != nil {
		return "", fmt.Errorf("error writing length field: %w", err)
	}

	w.Close()

	req, err := http.NewRequest("POST", "https://api.decopy.ai/api/decopy/ai-summarizer/create-job2", &b)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Origin", "https://decopy.ai")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Product-Code", "067003")
	req.Header.Set("Product-Serial", "b67ce40647796533eaa1d42b9fb5916e")
	req.Header.Set("Referer", "https://decopy.ai/")
	req.Header.Set("Sec-Ch-Ua", `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Gpc", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

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

	log.Debugf("Create job response: %s", string(body))

	var createResp CreateJobResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	if createResp.Code != 100000 {
		return "", fmt.Errorf("API returned error code: %v", createResp)
	}

	return createResp.Result.JobID, nil
}

func (r *MLRepository) getJobResult(jobID string) (string, error) {
	url := fmt.Sprintf("https://api.decopy.ai/api/decopy/ai-summarizer/get-job/%s", jobID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Origin", "https://decopy.ai")
	req.Header.Set("Referer", "https://decopy.ai/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")

			if dataStr == "Data transfer completed." {
				break
			}

			var sseData SSEData
			if err := json.Unmarshal([]byte(dataStr), &sseData); err != nil {
				log.Warnf("Failed to parse SSE data: %s, error: %v", dataStr, err)
				continue
			}

			if sseData.State == 100000 {
				result.WriteString(sseData.Data)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading SSE stream: %w", err)
	}

	content := result.String()
	if content == "" {
		return "", fmt.Errorf("no content received from API")
	}

	return cleanResponse(content), nil
}

func (r *MLRepository) SummarizeMessages(messages []string) (string, error) {
	combinedText := summaryPrompt + "\n\n"
	for _, msg := range messages {
		combinedText += msg + "\n\n"
	}

	log.Infof("Creating summarization job for %d messages", len(messages))

	jobID, err := r.createJob(combinedText)
	if err != nil {
		return "", fmt.Errorf("error creating job: %w", err)
	}

	log.Infof("Created job with ID: %s", jobID)

	result, err := r.getJobResult(jobID)
	if err != nil {
		return "", fmt.Errorf("error getting job result: %w", err)
	}

	log.Infof("Successfully received summarization result")

	return result, nil
}
