package config

import (
	"fmt"
	"os"
	"strings"
)

// Channels - список каналов для мониторинга
var Channels = map[int64]string{
	1429590454: "kontext_channel",
	1754252633: "topor_live",
}

const SessionDir = "session/telegram-session"

// Константы Storyline Tracking / TDT.
const (
	// Матчинг кандидатов-топиков к существующим сюжетам.
	// Пороги откалиброваны на проде 2026-07-22 скриптом scripts/threshold_calibration
	// (904 пары, разметка LLM-судьёй). Шкала — кросс-модельная косинусная близость
	// query-эмбеддинга кандидата к doc-эмбеддингу сюжета: у Яндекса она заметно ниже
	// doc-doc (медиана матчей ~0.48, не-матчей ~0.32).
	MatchSimHigh = 0.48 // sim >= HIGH -> авто-привязка без LLM (precision ~0.93)
	MatchSimLow  = 0.38 // sim < LOW -> новый сюжет; между LOW и HIGH -> LLM-подтверждение
	MatchTopK    = 5    // сколько ближайших сюжетов рассматривать на топик

	// Окно и пороги для метрик эскалации/шума.
	BaselineWindowDays = 14
	EscalationRatio    = 2.0
	EscalationMinCount = 3
	NoiseFreqFraction  = 0.6
	NoiseMaxImportance = 2

	// Жизненный цикл сюжетов.
	DormantAfterDays = 7
	ClosedAfterDays  = 30

	// Размерность эмбеддингов Yandex text-search-doc/query.
	EmbeddingDim = 256
)

// EmbedDocURI возвращает URI модели эмбеддинга документов.
// Использует YANDEX_EMBED_DOC_URI, иначе дефолт из folderID.
func EmbedDocURI(folderID string) string {
	if uri := strings.TrimSpace(os.Getenv("YANDEX_EMBED_DOC_URI")); uri != "" {
		return uri
	}
	return fmt.Sprintf("emb://%s/text-search-doc/latest", folderID)
}

// EmbedQueryURI возвращает URI модели эмбеддинга запросов.
// Использует YANDEX_EMBED_QUERY_URI, иначе дефолт из folderID.
func EmbedQueryURI(folderID string) string {
	if uri := strings.TrimSpace(os.Getenv("YANDEX_EMBED_QUERY_URI")); uri != "" {
		return uri
	}
	return fmt.Sprintf("emb://%s/text-search-query/latest", folderID)
}
