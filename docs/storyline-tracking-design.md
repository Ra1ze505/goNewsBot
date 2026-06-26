# Storyline Tracking / TDT — дизайн-документ

> Цель: научить суммаризацию различать **фоновый шум** (то, что происходит постоянно)
> от **нового события** и **эскалации** существующего, за счёт памяти о предыдущих днях.
>
> Подход: Topic Detection & Tracking — долгоживущие **сюжетные линии (storylines)**
> на канал, к которым каждый день привязываются новые сообщения; новизна/эскалация
> считается объективно по истории объёма и важности сюжета.
>
> Статус: дизайн. Кода ещё нет. Матчинг — на эмбеддингах + pgvector. Холодный старт —
> честный прогон пайплайна по дням за прошлую неделю на реальных сообщениях из БД.

---

## 1. Обзор и инварианты

Что НЕ меняем (чтобы не задеть пользовательские флоу):

- Таблица `summaries` остаётся источником финального текста дайджеста. `handlers/news.go`
  и `service/mailing.go` не трогаем — они по-прежнему читают `GetLatestSummary`.
- Гарантия «одна суммаризация на канал в день» (`HasSummaryToday`) сохраняется.
- Лимит длины Telegram (< 4096, у нас целимся в < 3500) сохраняется.

Что добавляем:

- Две таблицы: `storylines` (текущее состояние сюжета) и `storyline_observations`
  (дневной временной ряд по сюжету — основа для детекции эскалации).
- Расширение ML-слоя: извлечение топиков с категориями, эмбеддинги, подтверждение
  матчинга, рендер с группировкой.
- Новый `StorylineRepository`.
- Единый метод обработки одного дня `ProcessDay(channelID, date, messages)`, который
  переиспользуют и боевой путь (date = сегодня), и бэкфилл (date = исторические дни).
- Скрипт честного бэкфилла `scripts/storyline_backfill`.

---

## 2. Конвейер обработки одного дня

```
сообщения за день D (с реальными message_id)
      │
      ▼
[A] Извлечение топиков ─ LLM ─►  кандидаты {title, summary, category, importance, msg_ids}
      │
      ▼
[B] Эмбеддинг кандидатов ─ Yandex text-search-query (256d)
      │
      ▼
[C] Матчинг к сюжетам ─ pgvector top-K по active storylines ─►
      │   • sim ≥ HIGH                → авто-привязка
      │   • LOW ≤ sim < HIGH          → LLM подтверждает (id | NEW)
      │   • sim < LOW                 → NEW
      ▼
[D] Классификация ─ статистика из observations (< D) + LLM ─►
      │   change_type ∈ {new, escalation, ongoing, deescalation, recurring_noise}
      │   + delta_summary (что нового именно сегодня)
      ▼
[E] Обновление состояния ─ upsert storylines (state/importance/last_seen/embedding)
      │                     + insert storyline_observations(obs_date = D)
      │                     + mark dormant/closed по давности last_seen
      ▼
[F] Рендер дайджеста ─ LLM ─► группировка 🆕 / 🔺 / ▶️ + сворачивание фона
      │
      ▼
   summaries (как сейчас)
```

Стадии A и F эволюция существующих `extractTopicPlan` / `renderDigest`.
Новое — B, C, D, E.

---

## 3. Модель данных

Требуется расширение pgvector: `CREATE EXTENSION IF NOT EXISTS vector;`
(в Postgres должен быть установлен пакет pgvector; для прод-окружения — добавить в
инструкции деплоя/Docker).

```sql
-- db/migrations/0002_storylines.sql

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS storylines (
    id           SERIAL PRIMARY KEY,
    channel_id   BIGINT NOT NULL,
    title        TEXT   NOT NULL,                 -- каноничный заголовок (стабильный)
    state        TEXT   NOT NULL,                 -- "сводка обстановки": текущее состояние сюжета
    category     TEXT,                            -- рубрика: военное / происшествия / экономика / ...
    status       TEXT   NOT NULL DEFAULT 'active',-- active | dormant | closed
    importance   INT    NOT NULL DEFAULT 1,       -- актуальная важность 1..5
    embedding    vector(256),                     -- doc-эмбеддинг по title+state (Yandex text-search-doc)
    first_seen   DATE   NOT NULL,
    last_seen    DATE   NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_storylines_channel_status   ON storylines(channel_id, status);
CREATE INDEX IF NOT EXISTS idx_storylines_channel_lastseen ON storylines(channel_id, last_seen);

-- ANN-индекс для косинусного поиска. ivfflat требует ANALYZE/наполнения;
-- для малых объёмов можно начать без индекса (точный перебор) и добавить позже.
CREATE INDEX IF NOT EXISTS idx_storylines_embedding
    ON storylines USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE IF NOT EXISTS storyline_observations (
    id            SERIAL PRIMARY KEY,
    storyline_id  INT  NOT NULL REFERENCES storylines(id) ON DELETE CASCADE,
    channel_id    BIGINT NOT NULL,
    obs_date      DATE NOT NULL,
    message_count INT  NOT NULL DEFAULT 0,    -- сколько сообщений легло в сюжет в этот день
    importance    INT  NOT NULL DEFAULT 1,    -- важность сюжета в этот день
    change_type   TEXT NOT NULL,              -- new|escalation|ongoing|deescalation|recurring_noise
    delta_summary TEXT,                       -- что именно нового в этот день
    source_message_ids BIGINT[],              -- реальные message_id из messages
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (storyline_id, obs_date)           -- идемпотентность при перегенерации/бэкфилле
);

CREATE INDEX IF NOT EXISTS idx_obs_channel_date ON storyline_observations(channel_id, obs_date);
CREATE INDEX IF NOT EXISTS idx_obs_storyline    ON storyline_observations(storyline_id);
```

Заметки по хранению вектора (драйвер `lib/pq`, без ORM):

- Вектор пишем/читаем как pgvector-литерал строкой: `'[0.013,-0.024,...]'`.
- Можно добавить `github.com/pgvector/pgvector-go` (есть `pgvector.Vector` с поддержкой
  `database/sql` `Valuer`/`Scanner`), чтобы не форматировать вручную. Рекомендуется.
- Поиск: `ORDER BY embedding <=> $1 LIMIT $2`, где `<=>` — косинусная дистанция;
  `similarity = 1 - distance`.

---

## 4. Эмбеддинги (Yandex AI Studio)

- Модели асимметричные, обе 256-мерные:
  - `emb://<folder>/text-search-doc/latest` — для **документов** (состояние сюжета);
  - `emb://<folder>/text-search-query/latest` — для **запросов** (сегодняшний топик-кандидат).
- Эндпоинт OpenAI-совместимый: `POST {YANDEX_OPENAI_BASE_URL}/embeddings`
  (тот же базовый URL и токен-провайдер, что уже используются в `ml.go`).
- Через `openai-go`: `client.Embeddings.New(ctx, openai.EmbeddingNewParams{Model: docOrQueryURI, Input: ...})`.

Что чем кодируем:

- `storylines.embedding` = эмбеддинг `title + "\n" + state` моделью **text-search-doc**.
  Пересчитывается при каждом обновлении состояния сюжета.
- Кандидат-топик дня кодируется моделью **text-search-query** (`title + "\n" + summary`)
  и используется только для поиска ближайших сюжетов; не хранится.

Новые ENV (опциональны, есть дефолты):

- `YANDEX_EMBED_DOC_URI`  (default `emb://<folder>/text-search-doc/latest`)
- `YANDEX_EMBED_QUERY_URI`(default `emb://<folder>/text-search-query/latest`)

---

## 5. Матчинг (гибрид: вектор + LLM-подтверждение)

Алгоритм для каждого кандидата-топика дня:

1. Получить query-эмбеддинг кандидата.
2. Достать top-K active сюжетов канала по косинусу:
   `SELECT ... ORDER BY embedding <=> $q LIMIT K` (K = 5).
3. Решение по similarity (`sim = 1 - distance`):
   - `sim ≥ MATCH_SIM_HIGH` (0.80) → авто-привязка к лучшему сюжету.
   - `MATCH_SIM_LOW (0.55) ≤ sim < HIGH` → передать кандидата и top-K кратких карточек
     сюжетов в LLM, которая возвращает `id` или `NEW`.
   - `sim < MATCH_SIM_LOW` → новый сюжет.
4. Несколько сегодняшних топиков могут привязаться к одному сюжету — это нормально
   (агрегируем их `message_ids` и важности в одно наблюдение за день).

Почему гибрид: вектор даёт дешёвый отзыв (recall), LLM закрывает «серую зону»
(точность, precision) и не вызывается, когда всё очевидно. Для 2 каналов и единиц–
десятков активных сюжетов это и быстро, и дёшево.

Защита от фрагментации/над-слияния (вторая итерация):

- Периодический merge-шаг (раз в неделю): кластеризация active-сюжетов по косинусу +
  LLM-подтверждение слияния дублей.
- При слиянии: одна запись `storylines` остаётся канонической, наблюдения переносятся,
  вторая помечается `closed`.

---

## 6. Метрики эскалации и шума (объективная часть)

Для сюжета S на день D берём `storyline_observations` со `obs_date < D` за окно
`BASELINE_WINDOW_DAYS` (14):

- `baseline = median(message_count)` по окну;
- `volume_ratio = today_count / max(baseline, 1)`;
- `days_seen = число дней в окне, где сюжет встречался`;
- `median_importance` по окну.

Правила (rule-based `change_type`, считаются кодом ДО вызова LLM):

| Метка             | Условие |
|-------------------|---------|
| `new`             | `first_seen == D` (сюжета раньше не было) |
| `escalation`      | `volume_ratio ≥ ESCALATION_RATIO (2.0)` и `today_count ≥ ESCALATION_MIN_COUNT (3)`, **или** скачок важности `≥ +2` к `median_importance` |
| `recurring_noise` | `days_seen ≥ NOISE_FREQ_FRACTION (0.6) * window` и `median_importance ≤ NOISE_MAX_IMPORTANCE (2)` и `volume_ratio ≤ 1.3` |
| `deescalation`    | `volume_ratio ≤ 0.4` относительно baseline |
| `ongoing`         | иначе (есть содержательная новизна — определяет LLM в стадии D) |

Принцип: **числа считаем кодом и передаём LLM как факты** («сюжет встречался 6/14 дней,
медиана 3 сообщения/день, сегодня 4, важность 2»). LLM не выдумывает новизну, а лишь
формулирует качественную часть (`delta_summary`) на основе фактов. Это устраняет главную
слабость чисто-промптовых решений.

Диффузный шум (не связанные между собой ДТП и т.п., не образующие единый сюжет)
обрабатывается на уровне рубрик: высокочастотные низковажные `category` сворачиваются
в одну строку «фон как обычно» на стадии рендера. Лёгкий baseline по рубрикам — вторая
итерация.

---

## 7. Интерфейсы (Go)

### 7.1. Структуры данных

```go
// repository/storyline.go

type Storyline struct {
    ID         int64
    ChannelID  int64
    Title      string
    State      string
    Category   string
    Status     string    // active | dormant | closed
    Importance int
    Embedding  []float32  // 256
    FirstSeen  time.Time
    LastSeen   time.Time
}

type Observation struct {
    StorylineID      int64
    ChannelID        int64
    ObsDate          time.Time
    MessageCount     int
    Importance       int
    ChangeType       string
    DeltaSummary     string
    SourceMessageIDs []int64
}

// агрегаты, посчитанные из observations со obs_date < date
type StorylineStats struct {
    DaysSeen         int
    MedianCount      float64
    MedianImportance float64
    LastSeen         time.Time
}
```

### 7.2. StorylineRepository

```go
type StorylineRepositoryInterface interface {
    // матчинг
    SearchNearest(channelID int64, query []float32, k int) ([]ScoredStoryline, error) // active only
    GetActive(channelID int64) ([]Storyline, error)

    // статистика для классификации (строго obs_date < date)
    GetStats(storylineID int64, before time.Time, windowDays int) (StorylineStats, error)

    // запись состояния
    CreateStoryline(s *Storyline) (int64, error)
    UpdateStoryline(s *Storyline) error                 // state/title/importance/embedding/last_seen/status
    SaveObservation(o *Observation) error               // upsert по (storyline_id, obs_date)

    // жизненный цикл
    MarkDormant(channelID int64, lastSeenBefore time.Time) error
    MarkClosed(channelID int64, lastSeenBefore time.Time) error

    // идемпотентность перегенерации/бэкфилла
    DeleteObservationsForDate(channelID int64, date time.Time) error
    ResetChannel(channelID int64) error                  // для бэкфилла --reset
}

type ScoredStoryline struct {
    Storyline  Storyline
    Similarity float64
}
```

### 7.3. ML-слой (остаётся stateless, без БД)

```go
type MLRepositoryInterface interface {
    // стадия A
    ExtractTopics(messages []MessageInput) ([]CandidateTopic, error)
    // стадия B (эмбеддинги)
    EmbedDocuments(texts []string) ([][]float32, error) // text-search-doc
    EmbedQueries(texts []string) ([][]float32, error)   // text-search-query
    // стадия C: подтверждение в "серой зоне"
    ConfirmMatch(cand CandidateTopic, options []StorylineBrief) (matchedID int64, isNew bool, err error)
    // стадии D+F можно объединить, но для тестируемости разнесём:
    WriteDelta(in DeltaInput) (newState string, deltaSummary string, err error)
    RenderDigest(groups DigestGroups) (string, error)

    // обратная совместимость на время миграции (старый путь)
    SummarizeMessages(messages []string) (string, error)
}

type MessageInput struct {
    MessageID int64
    Text      string
}

type CandidateTopic struct {
    Title                string
    Summary              string
    Category             string
    Importance           int
    SourceMessageNumbers []int    // позиционные в дневной пачке
    SourceMessageIDs     []int64  // реальные, резолвятся из numbers
}

type StorylineBrief struct {
    ID         int64
    Title      string
    State      string
    LastSeen   string
    AvgCount   float64
    Similarity float64
}
```

> Чтобы заполнять `SourceMessageIDs`, нужно тянуть из БД пары `(message_id, text)`,
> а не только текст. Добавляем в `SummaryRepository` методы, возвращающие
> `[]MessageInput` (`GetMessagesForDateWithIDs`, `GetMessagesForLastDayWithIDs`),
> старые оставляем для совместимости.

---

## 8. Оркестрация (общий код для прод и бэкфилла)

Выносим обработку одного дня в единый метод, чтобы боевой путь и бэкфилл не разъезжались.

```go
// service/storyline.go
type StorylineProcessor struct {
    summaryRepo   repository.SummaryRepositoryInterface
    storylineRepo repository.StorylineRepositoryInterface
    mlRepo        repository.MLRepositoryInterface
}

// ProcessDay — A..F для одного (channelID, date).
// writeSummary=false в бэкфилле, если не нужно перезаписывать дневные дайджесты.
func (p *StorylineProcessor) ProcessDay(channelID int64, date time.Time, msgs []repository.MessageInput, writeSummary bool) (string, error)
```

Боевой `SummaryService.ProcessChannelSummaries(peerID)`:

1. `HasSummaryToday` → если есть, выходим (как сейчас).
2. Тянем сегодняшние сообщения с id.
3. `ProcessDay(peerID, today, msgs, writeSummary=true)`.
4. Пишем результат в `summaries` (как сейчас).

Граф зависимостей в `src/main.go` расширяем: добавляем `StorylineRepository` и
`StorylineProcessor`, прокидываем в `SummaryService`.

---

## 9. Промпты (черновики, рус.)

### 9.1. Извлечение топиков (A) — эволюция текущего
К текущему промпту добавляем поле `category` и просим возвращать `source_message_numbers`.

```
Ты аналитик новостной редакции. Преврати поток сообщений Telegram-канала в список топиков.
Правила: объединяй связанные сообщения; убирай рекламу/повторы/эмоции без фактов;
не больше 8 топиков; важность 1..5; укажи рубрику (военное/происшествия/экономика/политика/общество/другое);
сохрани номера исходных сообщений; не выдумывай факты. Верни только валидный JSON.
Схема: {"topics":[{"title","summary","category","importance","source_message_numbers":[..]}]}
```

### 9.2. Подтверждение матчинга (C, только серая зона)

```
Есть сегодняшний топик и несколько похожих существующих сюжетов канала.
Реши: топик — продолжение одного из сюжетов или это НОВЫЙ сюжет?
Не объединяй разные по сути сюжеты. Верни JSON: {"matched_id": <id|null>, "is_new": <bool>, "reason": "..."}.
ТОПИК: {title, summary}
СЮЖЕТЫ: [{id, title, state, last_seen, avg_count, similarity}]
```

### 9.3. Дельта + обновление состояния (D)

```
Дан сюжет (его текущее состояние) и сегодняшние сообщения по нему, плюс статистика.
Статистика (факты, не выдумывай иное): встречался {days_seen}/{window} дней,
медиана {median_count} сообщений/день, сегодня {today_count}, медианная важность {median_importance},
предварительная метка изменения: {change_type}.
Задача: 1) краткое "что нового именно сегодня" (delta_summary, 1-2 предложения, может быть пустым,
если новизны нет); 2) обновлённое состояние сюжета (state, до 600 символов, без воды).
Верни JSON: {"delta_summary": "...", "state": "..."}.
```

### 9.4. Рендер дайджеста (F)

```
Собери Telegram-дайджест по сгруппированным сюжетам. Группы и порядок:
🆕 Новое — сюжеты change_type=new;
🔺 Эскалация — change_type=escalation;
▶️ Развитие — change_type=ongoing/deescalation с непустым delta_summary.
recurring_noise НЕ перечисляй по одному — сверни в одну строку в конце:
"Фон без изменений: <рубрики/темы через запятую>".
Пиши по-русски, факты только из входных данных, без номеров сообщений, итог < 3500 символов.
```

---

## 10. Честный бэкфилл (прогон пайплайна по дням)

Не суммаризации задним числом, а **полноценный replay** конвейера по реальным
сообщениям из БД — так baseline и сюжеты к моменту запуска боевого режима будут настоящими.

Скрипт `scripts/storyline_backfill/main.go`:

Флаги:
- `--days N` (default 7) — сколько прошлых дней прогнать;
- `--end-date YYYY-MM-DD` (default вчера) — последний день replay;
- `--channel ID` (default — все из `config.Channels`);
- `--reset` (default true) — очистить `storylines`/`storyline_observations` канала перед прогоном;
- `--write-summaries` (default false) — писать ли дневной текст в `summaries`.

Логика:

```
для каждого канала:
    если --reset: storylineRepo.ResetChannel(channel)
    для D от (end-date - days + 1) до end-date  ВКЛЮЧИТЕЛЬНО, по возрастанию:
        msgs := summaryRepo.GetMessagesForDateWithIDs(channel, D)
        если msgs пуст: continue
        processor.ProcessDay(channel, D, msgs, writeSummary=--write-summaries)
        логируем: сколько сюжетов new/escalation/ongoing/noise за день
```

Ключевые свойства:
- Хронологический порядок ⇒ статистика накапливается естественно; метрика эскалации
  на день D видит только наблюдения со `obs_date < D` (мы их к этому моменту и вставили).
- Идемпотентность: `--reset` + `UNIQUE(storyline_id, obs_date)` делают повторный запуск безопасным.
- После бэкфилла боевой режим стартует с непустым baseline; первый «сегодня» уже
  корректно делит новости на новое/эскалацию/фон.

Бэкфилл переиспользует тот же `ProcessDay`, что и прод, — единый код, никакого дубля логики.

---

## 11. Краевые случаи и идемпотентность

- **Админ-регенерация** (`admin_handlers` + `DeleteLastSummary`): при TDT помимо удаления
  текста в `summaries` нужно откатить сегодняшние наблюдения
  (`DeleteObservationsForDate(channel, today)`) перед повторным `ProcessDay`, иначе
  двойной учёт объёма исказит baseline.
- **Двойной запуск за день** (сигнал `MessagesFetched` приходит несколько раз):
  `HasSummaryToday` гасит повтор боевого пути; на уровне наблюдений защищает
  `UNIQUE(storyline_id, obs_date)` + upsert.
- **Затухание**: после `DORMANT_AFTER_DAYS` (7) без `last_seen` → `dormant`
  (не участвует в матчинге/дайджесте); после `CLOSED_AFTER_DAYS` (30) → `closed`.
- **Дрейф состояния**: `state` ограничен ~600 символами; растущую историю держим в
  `storyline_observations.delta_summary`, а не в одном раздувающемся поле.
- **Рост контекста**: в матчинг и LLM-подтверждение идут только active-сюжеты, top-K.
- **Холодный pgvector ivfflat**: на старте мало данных — можно временно отключить
  ivfflat-индекс (точный перебор на десятках строк дёшев) и включить после бэкфилла.

---

## 12. Конфигурируемые константы (с дефолтами)

| Константа | Default | Назначение |
|-----------|---------|------------|
| `MATCH_SIM_HIGH` | 0.80 | авто-привязка без LLM |
| `MATCH_SIM_LOW` | 0.55 | ниже — новый сюжет |
| `MATCH_TOP_K` | 5 | сколько кандидатов-сюжетов на топик |
| `BASELINE_WINDOW_DAYS` | 14 | окно статистики |
| `ESCALATION_RATIO` | 2.0 | порог объёма для эскалации |
| `ESCALATION_MIN_COUNT` | 3 | минимум сообщений, чтобы считать эскалацией |
| `NOISE_FREQ_FRACTION` | 0.6 | доля дней присутствия для «шума» |
| `NOISE_MAX_IMPORTANCE` | 2 | потолок важности «шума» |
| `DORMANT_AFTER_DAYS` | 7 | перевод в dormant |
| `CLOSED_AFTER_DAYS` | 30 | перевод в closed |

Размещение: отдельный блок в `src/config` (или константы в `service/storyline.go`).

---

## 13. Фазирование внедрения

1. **Фундамент**: миграция `0002_storylines.sql` (+pgvector), `StorylineRepository`,
   методы `*WithIDs` в `SummaryRepository`, эмбеддинг-методы в ML-слое. Дайджест не меняется.
2. **Ядро TDT**: `ProcessDay` (A–E), матчинг (vector + LLM), статистика, апсерт состояния.
   Скрипт бэкфилла. Дайджест пока прежний.
3. **Рендер**: новый `RenderDigest` с группировкой 🆕/🔺/▶️ и сворачиванием фона —
   видимый эффект для пользователя.
4. **Робастность (опц.)**: merge-шаг против фрагментации, baseline по рубрикам,
   ANN-тюнинг ivfflat/hnsw.

---

## 14. Открытые вопросы / риски

- Поддержка pgvector в проде (Docker/managed Postgres) — нужно подтвердить наличие расширения.
- Стоимость эмбеддингов/вызовов LLM при бэкфилле (N дней × каналы × топики) — оценить лимиты Yandex.
- Калибровка порогов `MATCH_SIM_*` и метрик эскалации — потребует подгонки на реальных данных
  после первого бэкфилла (заложить лёгкое логирование решений матчинга для разбора).
- Тесты: репозиторий — sqlmock (включая pgvector-литералы); `ProcessDay`/ML — gomock
  по новым интерфейсам, как в текущем стиле.
```
