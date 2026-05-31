package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMLRepositoryUsesOpenAICompatibleDefaults(t *testing.T) {
	t.Setenv("YANDEX_FOLDER_ID", "test-folder")
	t.Setenv("YANDEX_API_KEY", "test-api-key")
	t.Setenv("YANDEX_SERVICE_ACCOUNT_KEY_PATH", "")
	t.Setenv("YANDEX_ASSISTANT_ID", "")
	t.Setenv("YANDEX_OPENAI_BASE_URL", "")
	t.Setenv("YANDEX_MODEL_URI", "")

	repo, err := NewMLRepository()

	require.NoError(t, err)
	assert.Equal(t, defaultYandexOpenAIBaseURL, repo.baseURL)
	assert.Equal(t, "test-folder", repo.folderID)
	assert.Equal(t, "gpt://test-folder/yandexgpt/latest", repo.modelURI)
	assert.NotEmpty(t, repo.systemPrompt)
}

func TestCreateChatCompletionSendsOpenAICompatibleRequest(t *testing.T) {
	var capturedRequest chatCompletionRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "test-folder", r.Header.Get("x-folder-id"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedRequest))

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Готовая сводка"}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := &MLRepository{
		httpClient:    server.Client(),
		tokenProvider: staticTokenProvider{token: "test-token"},
		baseURL:       server.URL,
		folderID:      "test-folder",
		modelURI:      "gpt://test-folder/yandexgpt/latest",
		systemPrompt:  newsSummarySystemPrompt,
		maxTokens:     defaultYandexMaxTokens,
		temperature:   defaultYandexTemperature,
	}

	result, err := repo.createChatCompletion(context.Background(), "Пользовательский промпт")

	require.NoError(t, err)
	assert.Equal(t, "Готовая сводка", result)
	assert.Equal(t, "gpt://test-folder/yandexgpt/latest", capturedRequest.Model)
	require.Len(t, capturedRequest.Messages, 2)
	assert.Equal(t, "system", capturedRequest.Messages[0].Role)
	assert.Contains(t, capturedRequest.Messages[0].Content, "Объединяй связанные сообщения")
	assert.Equal(t, "user", capturedRequest.Messages[1].Role)
	assert.Equal(t, "Пользовательский промпт", capturedRequest.Messages[1].Content)
	assert.Equal(t, defaultYandexMaxTokens, capturedRequest.MaxTokens)
	assert.Equal(t, defaultYandexTemperature, capturedRequest.Temperature)
}

func TestCreateChatCompletionReturnsAPIErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"error":{"message":"bad model uri"}}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := &MLRepository{
		httpClient:    server.Client(),
		tokenProvider: staticTokenProvider{token: "test-token"},
		baseURL:       server.URL,
		folderID:      "test-folder",
		modelURI:      "gpt://test-folder/yandexgpt/latest",
		systemPrompt:  newsSummarySystemPrompt,
		maxTokens:     defaultYandexMaxTokens,
		temperature:   defaultYandexTemperature,
	}

	result, err := repo.createChatCompletion(context.Background(), "prompt")

	assert.Empty(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad model uri")
}

func TestBuildSummaryUserPromptNumbersAndTrimsMessages(t *testing.T) {
	prompt := buildSummaryUserPrompt([]string{" первая новость ", "вторая новость"})

	assert.Contains(t, prompt, "[#1]\nпервая новость")
	assert.Contains(t, prompt, "[#2]\nвторая новость")
	assert.True(t, strings.HasPrefix(prompt, "Сгруппируй эти сообщения"))
}
