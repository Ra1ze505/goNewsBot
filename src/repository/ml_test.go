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

type testChatCompletionRequest struct {
	Model       string `json:"model"`
	MaxTokens   int    `json:"max_tokens"`
	Temperature float64
	Messages    []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

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
	assert.NotEmpty(t, repo.extractPrompt)
	assert.NotEmpty(t, repo.renderPrompt)
}

func TestCreateChatCompletionSendsOpenAICompatibleRequest(t *testing.T) {
	var capturedRequest testChatCompletionRequest

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
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		extractParams: chatCompletionParams{MaxTokens: defaultExtractMaxTokens, Temperature: defaultExtractTemperature},
		renderParams:  chatCompletionParams{MaxTokens: defaultRenderMaxTokens, Temperature: defaultRenderTemperature},
	}

	result, err := repo.createChatCompletion(context.Background(), repo.extractParams, topicExtractionSystemPrompt, "Пользовательский промпт")

	require.NoError(t, err)
	assert.Equal(t, "Готовая сводка", result)
	assert.Equal(t, "gpt://test-folder/yandexgpt/latest", capturedRequest.Model)
	require.Len(t, capturedRequest.Messages, 2)
	assert.Equal(t, "system", capturedRequest.Messages[0].Role)
	assert.Contains(t, capturedRequest.Messages[0].Content, "Объединяй связанные сообщения")
	assert.Equal(t, "user", capturedRequest.Messages[1].Role)
	assert.Equal(t, "Пользовательский промпт", capturedRequest.Messages[1].Content)
	assert.Equal(t, defaultExtractMaxTokens, capturedRequest.MaxTokens)
	assert.Equal(t, defaultExtractTemperature, capturedRequest.Temperature)
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
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		extractParams: chatCompletionParams{MaxTokens: defaultExtractMaxTokens, Temperature: defaultExtractTemperature},
		renderParams:  chatCompletionParams{MaxTokens: defaultRenderMaxTokens, Temperature: defaultRenderTemperature},
	}

	result, err := repo.createChatCompletion(context.Background(), repo.extractParams, topicExtractionSystemPrompt, "prompt")

	assert.Empty(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad model uri")
}

func TestSummarizeMessagesExtractsTopicsThenRendersDigest(t *testing.T) {
	var requests []testChatCompletionRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request testChatCompletionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		requests = append(requests, request)

		w.Header().Set("Content-Type", "application/json")
		if len(requests) == 1 {
			response, err := json.Marshal(map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": "```json\n{\"topics\":[{\"title\":\"Курс рубля\",\"summary\":\"Рубль ослаб на фоне решений регулятора.\",\"why_important\":\"Это влияет на цены и импорт.\",\"importance\":5,\"source_message_numbers\":[1,2,99]},{\"title\":\"Локальный шум\",\"summary\":\"Малозначимое сообщение.\",\"importance\":1,\"source_message_numbers\":[3]}],\"noise_message_numbers\":[3,3,42]}\n```",
						},
					},
				},
			})
			require.NoError(t, err)
			_, err = w.Write(response)
			require.NoError(t, err)
			return
		}

		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"**1. Курс рубля**\nРубль ослаб на фоне решений регулятора.\nПочему важно: это влияет на цены и импорт."}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := &MLRepository{
		httpClient:    server.Client(),
		tokenProvider: staticTokenProvider{token: "test-token"},
		baseURL:       server.URL,
		folderID:      "test-folder",
		modelURI:      "gpt://test-folder/yandexgpt/latest",
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		extractParams: chatCompletionParams{MaxTokens: defaultExtractMaxTokens, Temperature: defaultExtractTemperature},
		renderParams:  chatCompletionParams{MaxTokens: defaultRenderMaxTokens, Temperature: defaultRenderTemperature},
	}

	result, err := repo.SummarizeMessages([]string{
		"Рубль снизился к доллару.",
		"ЦБ прокомментировал ситуацию на валютном рынке.",
		"Реклама канала.",
	})

	require.NoError(t, err)
	assert.Contains(t, result, "**1. Курс рубля**")
	require.Len(t, requests, 2)
	assert.Equal(t, "system", requests[0].Messages[0].Role)
	assert.Contains(t, requests[0].Messages[0].Content, "структурированный план")
	assert.Contains(t, requests[0].Messages[1].Content, "[#1]\nРубль снизился к доллару.")
	assert.Equal(t, "system", requests[1].Messages[0].Role)
	assert.Contains(t, requests[1].Messages[0].Content, "финальную сводку")
	assert.Contains(t, requests[1].Messages[1].Content, `"title": "Курс рубля"`)
	assert.NotContains(t, requests[1].Messages[1].Content, "99")
	assert.Contains(t, requests[1].Messages[1].Content, `"noise_message_numbers": [`)
	assert.Equal(t, defaultExtractMaxTokens, requests[0].MaxTokens)
	assert.Equal(t, defaultExtractTemperature, requests[0].Temperature)
	assert.Equal(t, defaultRenderMaxTokens, requests[1].MaxTokens)
	assert.Equal(t, defaultRenderTemperature, requests[1].Temperature)
}

func TestSummarizeMessagesReturnsEmptyDigestWhenNoTopics(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"topics\":[],\"noise_message_numbers\":[1]}"}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := &MLRepository{
		httpClient:    server.Client(),
		tokenProvider: staticTokenProvider{token: "test-token"},
		baseURL:       server.URL,
		folderID:      "test-folder",
		modelURI:      "gpt://test-folder/yandexgpt/latest",
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		extractParams: chatCompletionParams{MaxTokens: defaultExtractMaxTokens, Temperature: defaultExtractTemperature},
		renderParams:  chatCompletionParams{MaxTokens: defaultRenderMaxTokens, Temperature: defaultRenderTemperature},
	}

	result, err := repo.SummarizeMessages([]string{"Рекламный пост"})

	require.NoError(t, err)
	assert.Equal(t, "За последние сутки значимых новостей не найдено.", result)
	assert.Equal(t, 1, requestCount)
}

func TestBuildTopicExtractionPromptNumbersAndTrimsMessages(t *testing.T) {
	prompt := buildTopicExtractionPrompt([]string{" первая новость ", "вторая новость"})

	assert.Contains(t, prompt, "[#1]\nпервая новость")
	assert.Contains(t, prompt, "[#2]\nвторая новость")
	assert.True(t, strings.HasPrefix(prompt, "Проанализируй сообщения"))
}

func newTestMLRepo(server *httptest.Server) *MLRepository {
	return &MLRepository{
		httpClient:    server.Client(),
		tokenProvider: staticTokenProvider{token: "test-token"},
		baseURL:       server.URL + "/",
		folderID:      "test-folder",
		modelURI:      "gpt://test-folder/yandexgpt/latest",
		embedDocURI:   "emb://test-folder/text-search-doc/latest",
		embedQueryURI: "emb://test-folder/text-search-query/latest",
		extractPrompt: topicExtractionSystemPrompt,
		renderPrompt:  digestRenderSystemPrompt,
		extractParams: chatCompletionParams{MaxTokens: defaultExtractMaxTokens, Temperature: defaultExtractTemperature},
		renderParams:  chatCompletionParams{MaxTokens: defaultRenderMaxTokens, Temperature: defaultRenderTemperature},
	}
}

func TestExtractTopicsParsesCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"topics\":[{\"title\":\"Тема\",\"summary\":\"Суть\",\"category\":\"экономика\",\"importance\":9,\"source_message_numbers\":[1,2,99]}]}"}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := newTestMLRepo(server)

	candidates, err := repo.ExtractTopics([]MessageInput{{MessageID: 10, Text: "a"}, {MessageID: 11, Text: "b"}})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "Тема", candidates[0].Title)
	assert.Equal(t, "экономика", candidates[0].Category)
	assert.Equal(t, 5, candidates[0].Importance)
	assert.Equal(t, []int{1, 2}, candidates[0].SourceMessageNumbers) // 99 отброшен
}

func TestEmbedQueriesReturnsFloat32(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/embeddings", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"data":[{"embedding":[0.5,-0.25,1.0],"index":0,"object":"embedding"}],"model":"m","object":"list"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := newTestMLRepo(server)

	vecs, err := repo.EmbedQueries([]string{"запрос"})
	require.NoError(t, err)
	require.Len(t, vecs, 1)
	assert.Equal(t, []float32{0.5, -0.25, 1.0}, vecs[0])
}

func TestConfirmMatchValidatesID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"matched_id\":42,\"is_new\":false,\"reason\":\"продолжение\"}"}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := newTestMLRepo(server)

	matchedID, isNew, err := repo.ConfirmMatch(
		CandidateTopic{Title: "T", Summary: "S"},
		[]StorylineBrief{{ID: 42, Title: "Сюжет"}},
	)
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, int64(42), matchedID)
}

func TestConfirmMatchUnknownIDFallsBackToNew(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"matched_id\":999,\"is_new\":false}"}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := newTestMLRepo(server)

	_, isNew, err := repo.ConfirmMatch(CandidateTopic{Title: "T"}, []StorylineBrief{{ID: 42}})
	require.NoError(t, err)
	assert.True(t, isNew)
}

func TestWriteDeltaFallsBackToCurrentState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"delta_summary\":\"новое\",\"state\":\"\"}"}}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	repo := newTestMLRepo(server)

	state, delta, err := repo.WriteDelta(DeltaInput{Title: "T", CurrentState: "старое состояние", ChangeType: "ongoing"})
	require.NoError(t, err)
	assert.Equal(t, "старое состояние", state)
	assert.Equal(t, "новое", delta)
}

func TestRenderDigestEmptyGroups(t *testing.T) {
	repo := &MLRepository{}
	result, err := repo.RenderDigest(DigestGroups{})
	require.NoError(t, err)
	assert.Equal(t, "За последние сутки значимых новостей не найдено.", result)
}

func TestParseSummaryTopicPlanNormalizesTopics(t *testing.T) {
	topicPlan, err := parseSummaryTopicPlan(`{
		"topics": [
			{"title":"Средняя важность","summary":"Описание","importance":3,"source_message_numbers":[2,2,7]},
			{"title":"  Высокая важность  ","summary":"  Описание  ","importance":9,"source_message_numbers":[1]},
			{"title":"","summary":"Без заголовка","importance":5,"source_message_numbers":[3]}
		],
		"noise_message_numbers":[3,3,9]
	}`, 3)

	require.NoError(t, err)
	require.Len(t, topicPlan.Topics, 2)
	assert.Equal(t, "Высокая важность", topicPlan.Topics[0].Title)
	assert.Equal(t, 5, topicPlan.Topics[0].Importance)
	assert.Equal(t, []int{1}, topicPlan.Topics[0].SourceMessageNumbers)
	assert.Equal(t, "Средняя важность", topicPlan.Topics[1].Title)
	assert.Equal(t, []int{2}, topicPlan.Topics[1].SourceMessageNumbers)
	assert.Equal(t, []int{3}, topicPlan.NoiseMessageNumbers)
}
