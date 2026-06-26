package service

import (
	"fmt"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
)

// StorylineProcessor реализует единый конвейер обработки одного дня (A–F),
// который переиспользуют боевой путь и бэкфилл.
type StorylineProcessor struct {
	summaryRepo   repository.SummaryRepositoryInterface
	storylineRepo repository.StorylineRepositoryInterface
	mlRepo        repository.MLRepositoryInterface
}

func NewStorylineProcessor(
	summaryRepo repository.SummaryRepositoryInterface,
	storylineRepo repository.StorylineRepositoryInterface,
	mlRepo repository.MLRepositoryInterface,
) *StorylineProcessor {
	return &StorylineProcessor{
		summaryRepo:   summaryRepo,
		storylineRepo: storylineRepo,
		mlRepo:        mlRepo,
	}
}

// aggregation - группа сегодняшних кандидатов, привязанных к одному сюжету.
type aggregation struct {
	existing   *repository.Storyline // nil => новый сюжет
	candidates []repository.CandidateTopic
}

// digestEntry - результат обработки одного сюжета за день, для рендера.
type digestEntry struct {
	title        string
	deltaSummary string
	state        string
	category     string
	importance   int
	changeType   string
}

// ProcessDay выполняет стадии A–F для одного (channelID, date) и возвращает текст дайджеста.
// Запись в summaries остаётся вызывающему.
func (p *StorylineProcessor) ProcessDay(channelID int64, date time.Time, msgs []repository.MessageInput) (string, error) {
	day := truncateToDay(date)

	// A: извлечение топиков.
	candidates, err := p.mlRepo.ExtractTopics(msgs)
	if err != nil {
		return "", fmt.Errorf("failed to extract topics: %w", err)
	}
	if len(candidates) == 0 {
		log.Infof("No topics extracted for channel %d on %s", channelID, day.Format("2006-01-02"))
		return p.mlRepo.RenderDigest(repository.DigestGroups{})
	}

	// Резолв позиционных номеров в реальные message_id и индексы текста.
	idByPos := make(map[int]int64, len(msgs))
	textByID := make(map[int64]string, len(msgs))
	for i, m := range msgs {
		idByPos[i+1] = m.MessageID
		textByID[m.MessageID] = m.Text
	}

	queryTexts := make([]string, len(candidates))
	for i := range candidates {
		queryTexts[i] = candidates[i].Title + "\n" + candidates[i].Summary
		ids := make([]int64, 0, len(candidates[i].SourceMessageNumbers))
		for _, n := range candidates[i].SourceMessageNumbers {
			if id, ok := idByPos[n]; ok {
				ids = append(ids, id)
			}
		}
		candidates[i].SourceMessageIDs = ids
	}

	// B: эмбеддинги кандидатов (query-модель).
	qvecs, err := p.mlRepo.EmbedQueries(queryTexts)
	if err != nil {
		return "", fmt.Errorf("failed to embed candidate topics: %w", err)
	}
	if len(qvecs) != len(candidates) {
		return "", fmt.Errorf("embedding count mismatch: got %d for %d candidates", len(qvecs), len(candidates))
	}

	// C: матчинг и агрегация по сюжетам.
	existingAgg := make(map[int64]*aggregation)
	var newAgg []*aggregation
	for i := range candidates {
		matched, err := p.matchCandidate(channelID, day, candidates[i], qvecs[i])
		if err != nil {
			return "", err
		}
		if matched == nil {
			newAgg = append(newAgg, &aggregation{candidates: []repository.CandidateTopic{candidates[i]}})
			continue
		}
		agg, ok := existingAgg[matched.ID]
		if !ok {
			agg = &aggregation{existing: matched}
			existingAgg[matched.ID] = agg
		}
		agg.candidates = append(agg.candidates, candidates[i])
	}

	// D+E: классификация, дельта, апсерт состояния и наблюдений.
	var entries []digestEntry
	for _, agg := range existingAgg {
		entry, err := p.processStoryline(channelID, day, agg, textByID)
		if err != nil {
			return "", err
		}
		entries = append(entries, entry)
	}
	for _, agg := range newAgg {
		entry, err := p.processStoryline(channelID, day, agg, textByID)
		if err != nil {
			return "", err
		}
		entries = append(entries, entry)
	}

	// Жизненный цикл сюжетов по давности last_seen.
	if err := p.storylineRepo.MarkDormant(channelID, day.AddDate(0, 0, -config.DormantAfterDays)); err != nil {
		return "", fmt.Errorf("failed to mark dormant storylines: %w", err)
	}
	if err := p.storylineRepo.MarkClosed(channelID, day.AddDate(0, 0, -config.ClosedAfterDays)); err != nil {
		return "", fmt.Errorf("failed to mark closed storylines: %w", err)
	}

	// F: рендер сгруппированного дайджеста.
	return p.mlRepo.RenderDigest(buildDigestGroups(entries))
}

// matchCandidate возвращает существующий сюжет для привязки или nil для нового.
func (p *StorylineProcessor) matchCandidate(channelID int64, day time.Time, cand repository.CandidateTopic, qvec []float32) (*repository.Storyline, error) {
	scored, err := p.storylineRepo.SearchNearest(channelID, qvec, config.MatchTopK)
	if err != nil {
		return nil, fmt.Errorf("failed to search nearest storylines: %w", err)
	}
	if len(scored) == 0 {
		return nil, nil
	}

	best := scored[0]
	if best.Similarity >= config.MatchSimHigh {
		s := best.Storyline
		return &s, nil
	}
	if best.Similarity < config.MatchSimLow {
		return nil, nil
	}

	// Серая зона: подтверждаем матчинг через LLM.
	briefs := make([]repository.StorylineBrief, 0, len(scored))
	byID := make(map[int64]repository.Storyline, len(scored))
	for _, sc := range scored {
		stats, err := p.storylineRepo.GetStats(sc.Storyline.ID, day, config.BaselineWindowDays)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats for storyline %d: %w", sc.Storyline.ID, err)
		}
		briefs = append(briefs, repository.StorylineBrief{
			ID:         sc.Storyline.ID,
			Title:      sc.Storyline.Title,
			State:      sc.Storyline.State,
			LastSeen:   sc.Storyline.LastSeen.Format("2006-01-02"),
			AvgCount:   stats.MedianCount,
			Similarity: sc.Similarity,
		})
		byID[sc.Storyline.ID] = sc.Storyline
	}

	matchedID, isNew, err := p.mlRepo.ConfirmMatch(cand, briefs)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm match: %w", err)
	}
	if isNew {
		return nil, nil
	}
	if s, ok := byID[matchedID]; ok {
		return &s, nil
	}
	return nil, nil
}

// processStoryline обрабатывает один сюжет (новый или существующий): D+E.
func (p *StorylineProcessor) processStoryline(channelID int64, day time.Time, agg *aggregation, textByID map[int64]string) (digestEntry, error) {
	isNew := agg.existing == nil

	// Агрегируем источники, важность, рубрику.
	seen := make(map[int64]struct{})
	var sourceIDs []int64
	todayImportance := 1
	category := ""
	for _, c := range agg.candidates {
		for _, id := range c.SourceMessageIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			sourceIDs = append(sourceIDs, id)
		}
		if c.Importance > todayImportance {
			todayImportance = c.Importance
		}
		if category == "" && c.Category != "" {
			category = c.Category
		}
	}

	todayCount := len(sourceIDs)
	if todayCount == 0 {
		todayCount = len(agg.candidates)
	}

	todayMessages := make([]string, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		if txt, ok := textByID[id]; ok && txt != "" {
			todayMessages = append(todayMessages, txt)
		}
	}
	if len(todayMessages) == 0 {
		for _, c := range agg.candidates {
			todayMessages = append(todayMessages, c.Title+": "+c.Summary)
		}
	}

	title := agg.candidates[0].Title
	currentState := ""
	var stats repository.StorylineStats
	changeType := "new"
	if !isNew {
		title = agg.existing.Title
		currentState = agg.existing.State
		var err error
		stats, err = p.storylineRepo.GetStats(agg.existing.ID, day, config.BaselineWindowDays)
		if err != nil {
			return digestEntry{}, fmt.Errorf("failed to get stats for storyline %d: %w", agg.existing.ID, err)
		}
		changeType = classifyChangeType(stats, todayCount, todayImportance)
		if category == "" {
			category = agg.existing.Category
		}
	}

	// D: дельта + обновлённое состояние.
	newState, deltaSummary, err := p.mlRepo.WriteDelta(repository.DeltaInput{
		Title:            title,
		CurrentState:     currentState,
		TodayCount:       todayCount,
		DaysSeen:         stats.DaysSeen,
		WindowDays:       config.BaselineWindowDays,
		MedianCount:      stats.MedianCount,
		MedianImportance: stats.MedianImportance,
		ChangeType:       changeType,
		TodayMessages:    todayMessages,
	})
	if err != nil {
		return digestEntry{}, fmt.Errorf("failed to write delta: %w", err)
	}

	// E: эмбеддинг состояния + апсерт сюжета.
	embeddings, err := p.mlRepo.EmbedDocuments([]string{title + "\n" + newState})
	if err != nil {
		return digestEntry{}, fmt.Errorf("failed to embed storyline state: %w", err)
	}
	if len(embeddings) == 0 {
		return digestEntry{}, fmt.Errorf("received empty document embedding")
	}
	embedding := embeddings[0]

	var storylineID int64
	if isNew {
		storylineID, err = p.storylineRepo.CreateStoryline(&repository.Storyline{
			ChannelID:  channelID,
			Title:      title,
			State:      newState,
			Category:   category,
			Status:     "active",
			Importance: todayImportance,
			Embedding:  embedding,
			FirstSeen:  day,
			LastSeen:   day,
		})
		if err != nil {
			return digestEntry{}, fmt.Errorf("failed to create storyline: %w", err)
		}
	} else {
		storylineID = agg.existing.ID
		if err := p.storylineRepo.UpdateStoryline(&repository.Storyline{
			ID:         storylineID,
			ChannelID:  channelID,
			Title:      title,
			State:      newState,
			Category:   category,
			Status:     "active",
			Importance: todayImportance,
			Embedding:  embedding,
			LastSeen:   day,
		}); err != nil {
			return digestEntry{}, fmt.Errorf("failed to update storyline %d: %w", storylineID, err)
		}
	}

	if err := p.storylineRepo.SaveObservation(&repository.Observation{
		StorylineID:      storylineID,
		ChannelID:        channelID,
		ObsDate:          day,
		MessageCount:     todayCount,
		Importance:       todayImportance,
		ChangeType:       changeType,
		DeltaSummary:     deltaSummary,
		SourceMessageIDs: sourceIDs,
	}); err != nil {
		return digestEntry{}, fmt.Errorf("failed to save observation for storyline %d: %w", storylineID, err)
	}

	return digestEntry{
		title:        title,
		deltaSummary: deltaSummary,
		state:        newState,
		category:     category,
		importance:   todayImportance,
		changeType:   changeType,
	}, nil
}

// classifyChangeType реализует rule-based детекцию по дизайну §6.
func classifyChangeType(stats repository.StorylineStats, todayCount, todayImportance int) string {
	window := float64(config.BaselineWindowDays)
	baseline := stats.MedianCount
	if baseline < 1 {
		baseline = 1
	}
	volumeRatio := float64(todayCount) / baseline

	if (volumeRatio >= config.EscalationRatio && todayCount >= config.EscalationMinCount) ||
		(float64(todayImportance)-stats.MedianImportance >= 2) {
		return "escalation"
	}
	if float64(stats.DaysSeen) >= config.NoiseFreqFraction*window &&
		stats.MedianImportance <= float64(config.NoiseMaxImportance) &&
		volumeRatio <= 1.3 {
		return "recurring_noise"
	}
	if volumeRatio <= 0.4 {
		return "deescalation"
	}
	return "ongoing"
}

// buildDigestGroups раскладывает обработанные сюжеты по группам рендера.
func buildDigestGroups(entries []digestEntry) repository.DigestGroups {
	var groups repository.DigestGroups
	noiseSeen := make(map[string]struct{})
	for _, e := range entries {
		item := repository.DigestItem{
			Title:        e.title,
			DeltaSummary: e.deltaSummary,
			State:        e.state,
			Category:     e.category,
			Importance:   e.importance,
		}
		switch e.changeType {
		case "new":
			groups.New = append(groups.New, item)
		case "escalation":
			groups.Escalation = append(groups.Escalation, item)
		case "ongoing", "deescalation":
			if e.deltaSummary != "" {
				groups.Ongoing = append(groups.Ongoing, item)
			}
		case "recurring_noise":
			label := e.category
			if label == "" {
				label = e.title
			}
			if _, ok := noiseSeen[label]; !ok {
				noiseSeen[label] = struct{}{}
				groups.RecurringNoise = append(groups.RecurringNoise, label)
			}
		}
	}
	return groups
}

func truncateToDay(date time.Time) time.Time {
	u := date.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
