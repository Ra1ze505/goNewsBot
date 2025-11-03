package repository

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/ai/assistants/v1/runs"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/ai/assistants/v1/threads"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	log "github.com/sirupsen/logrus"
)

const (
	// Yandex Cloud AI Assistants API endpoint
	apiEndpoint = "assistant.api.cloud.yandex.net:443"
)

// YandexClient представляет клиент для работы с Yandex Cloud AI Assistants
type YandexClient struct {
	conn     *grpc.ClientConn
	sdk      *ycsdk.SDK
	folderID string
}

// ThreadManager управляет тредами и сообщениями
type ThreadManager struct {
	client         *YandexClient
	threadsClient  threads.ThreadServiceClient
	messagesClient threads.MessageServiceClient
	runsClient     runs.RunServiceClient
}

// NewYandexClient создает новый Yandex Cloud клиент
func NewYandexClient(ctx context.Context, serviceAccountKeyPath, folderID string) (*YandexClient, error) {
	// Чтение ключа сервисного аккаунта
	keyData, err := os.ReadFile(serviceAccountKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account key: %w", err)
	}

	// Парсинг ключа
	key, err := iamkey.ReadFromJSONBytes(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account key: %w", err)
	}

	// Создание credentials
	creds, err := ycsdk.ServiceAccountKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	// Создание SDK с аутентификацией через сервисный аккаунт
	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: creds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex Cloud SDK: %w", err)
	}

	// Создание gRPC соединения для Assistants API
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	// Создаем перехватчик для добавления IAM токена к унарным запросам
	unaryInterceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Получаем IAM токен через SDK
		token, err := sdk.CreateIAMToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get IAM token: %w", err)
		}

		md := metadata.Pairs(
			"authorization", "Bearer "+token.IamToken,
			"x-folder-id", folderID,
		)
		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(ctx, method, req, reply, cc, opts...)
	}

	// Создаем перехватчик для добавления IAM токена к потоковым запросам
	streamInterceptor := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Получаем IAM токен через SDK
		token, err := sdk.CreateIAMToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get IAM token: %w", err)
		}

		md := metadata.Pairs(
			"authorization", "Bearer "+token.IamToken,
			"x-folder-id", folderID,
		)
		ctx = metadata.NewOutgoingContext(ctx, md)

		return streamer(ctx, desc, cc, method, opts...)
	}

	conn, err := grpc.NewClient(
		apiEndpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithUnaryInterceptor(unaryInterceptor),
		grpc.WithStreamInterceptor(streamInterceptor),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	return &YandexClient{
		conn:     conn,
		sdk:      sdk,
		folderID: folderID,
	}, nil
}

// FolderID возвращает ID папки
func (c *YandexClient) FolderID() string {
	return c.folderID
}

// Conn возвращает gRPC соединение
func (c *YandexClient) Conn() *grpc.ClientConn {
	return c.conn
}

// SDK возвращает Yandex Cloud SDK
func (c *YandexClient) SDK() *ycsdk.SDK {
	return c.sdk
}

// Close закрывает соединение
func (c *YandexClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// NewThreadManager создает новый менеджер тредов
func NewThreadManager(client *YandexClient) (*ThreadManager, error) {
	conn := client.Conn()

	return &ThreadManager{
		client:         client,
		threadsClient:  threads.NewThreadServiceClient(conn),
		messagesClient: threads.NewMessageServiceClient(conn),
		runsClient:     runs.NewRunServiceClient(conn),
	}, nil
}

// CreateThread создает новый тред
func (tm *ThreadManager) CreateThread(ctx context.Context) (string, error) {
	req := &threads.CreateThreadRequest{
		FolderId: tm.client.FolderID(),
	}

	resp, err := tm.threadsClient.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create thread (folder_id=%s): %w", tm.client.FolderID(), err)
	}

	return resp.Id, nil
}

// AddMessage добавляет сообщение в тред
func (tm *ThreadManager) AddMessage(ctx context.Context, threadID, text string) error {
	req := &threads.CreateMessageRequest{
		ThreadId: threadID,
		Content: &threads.MessageContent{
			Content: []*threads.ContentPart{
				{
					PartType: &threads.ContentPart_Text{
						Text: &threads.Text{
							Content: text,
						},
					},
				},
			},
		},
	}

	_, err := tm.messagesClient.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}

// RunAssistant запускает ассистента и возвращает ответ
func (tm *ThreadManager) RunAssistant(ctx context.Context, threadID, assistantID string) (string, error) {
	// Сначала создаем Run
	createReq := &runs.CreateRunRequest{
		AssistantId: assistantID,
		ThreadId:    threadID,
		Stream:      true,
	}

	log.Debugf("[RUN] Creating run for thread=%s, assistant=%s", threadID, assistantID)
	run, err := tm.runsClient.Create(ctx, createReq)
	if err != nil {
		log.Errorf("[RUN] ERROR: Failed to create run: %v", err)
		return "", fmt.Errorf("failed to create run: %w", err)
	}

	runID := run.GetId()
	log.Debugf("[RUN] Successfully created run: run_id=%s", runID)

	// Подключаемся к стриму событий
	listenReq := &runs.ListenRunRequest{
		RunId: runID,
	}

	log.Debugf("[RUN] Starting to listen to run events for run_id=%s", runID)
	stream, err := tm.runsClient.Listen(ctx, listenReq)
	if err != nil {
		log.Errorf("[RUN] ERROR: Failed to listen to run: %v", err)
		return "", fmt.Errorf("failed to listen to run: %w", err)
	}

	// Собираем ответ из стрима
	var responseBuilder strings.Builder
	eventCount := 0

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			log.Debugf("[RUN] Stream ended normally after %d events", eventCount)
			break
		}
		if err != nil {
			log.Errorf("[RUN] ERROR: Failed to receive stream event: %v", err)
			return "", fmt.Errorf("failed to receive stream event: %w", err)
		}

		eventCount++

		// Обрабатываем разные типы событий
		switch e := event.EventData.(type) {
		case *runs.StreamEvent_PartialMessage:
			// Частичное сообщение - только логируем для отладки, не добавляем к ответу
			if e.PartialMessage != nil && e.PartialMessage.Content != nil && len(e.PartialMessage.Content) > 0 {
				totalLength := 0
				for _, part := range e.PartialMessage.Content {
					if textPart := part.GetText(); textPart != nil {
						totalLength += len(textPart.GetContent())
					}
				}
				log.Debugf("[RUN] [PARTIAL_MESSAGE] Event %d: received partial message (total length=%d)", eventCount, totalLength)
			} else {
				log.Debugf("[RUN] [PARTIAL_MESSAGE] Event %d: empty or missing content", eventCount)
			}

		case *runs.StreamEvent_CompletedMessage:
			// Полное сообщение - используем только его (оно содержит весь текст целиком)
			if e.CompletedMessage != nil && e.CompletedMessage.Content != nil && len(e.CompletedMessage.Content.Content) > 0 {
				// Очищаем предыдущий текст (на случай если что-то было)
				responseBuilder.Reset()
				partCount := 0
				for _, part := range e.CompletedMessage.Content.Content {
					if textPart := part.GetText(); textPart != nil {
						content := textPart.GetContent()
						responseBuilder.WriteString(content)
						partCount++
						log.Debugf("[RUN] [COMPLETED_MESSAGE] Event %d: received text part (length=%d)", eventCount, len(content))
					}
				}
				if partCount == 0 {
					log.Debugf("[RUN] [COMPLETED_MESSAGE] Event %d: no text parts found", eventCount)
				}
			} else {
				log.Debugf("[RUN] [COMPLETED_MESSAGE] Event %d: empty or missing content", eventCount)
			}

		case *runs.StreamEvent_Error:
			// Ошибка при выполнении
			if e.Error != nil {
				log.Errorf("[RUN] [ERROR] Event %d: assistant error - code=%d, message=%s", eventCount, e.Error.GetCode(), e.Error.GetMessage())
				return "", fmt.Errorf("assistant error: %d - %s", e.Error.GetCode(), e.Error.GetMessage())
			}

		default:
			log.Debugf("[RUN] Event %d: unknown event type %T", eventCount, e)
		}
	}

	response := responseBuilder.String()
	if response == "" {
		log.Errorf("[RUN] ERROR: Received empty response from assistant after %d events", eventCount)
		return "", fmt.Errorf("received empty response from assistant")
	}

	log.Debugf("[RUN] SUCCESS: Received complete response from assistant (length=%d, events=%d)", len(response), eventCount)
	return response, nil
}
