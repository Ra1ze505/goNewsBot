package repository

import (
	"context"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	log "github.com/sirupsen/logrus"
)

type SummaryService struct {
	summaryRepo SummaryRepositoryInterface
	mlService   *MLService
}

func NewSummaryService(summaryRepo SummaryRepositoryInterface, mlService *MLService) *SummaryService {
	return &SummaryService{
		summaryRepo: summaryRepo,
		mlService:   mlService,
	}
}

func (s *SummaryService) StartSummaryFetcher(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	if err := s.processAllChannels(); err != nil {
		log.Errorf("Error processing summaries on startup: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processAllChannels(); err != nil {
				log.Errorf("Error processing summaries: %v", err)
			}
		}
	}
}

func (s *SummaryService) processAllChannels() error {
	for _, channelUsername := range config.Channels {
		if err := s.ProcessChannelSummaries(channelUsername); err != nil {
			log.Errorf("Error processing summary for channel %s: %v", channelUsername, err)
			continue
		}
	}
	log.Info("Successfully processed summaries for all channels")
	return nil
}

func (s *SummaryService) ProcessChannelSummaries(channelUsername string) error {
	channelID, err := s.summaryRepo.GetChannelID(channelUsername)
	if err != nil {
		return err
	}
	if channelID == 0 {
		return nil
	}

	hasSummary, err := s.summaryRepo.HasSummaryToday(channelID)
	if err != nil {
		return err
	}
	if hasSummary {
		return nil
	}

	messages, err := s.summaryRepo.GetMessagesForLastDay(channelID)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		return nil
	}

	summary, err := s.mlService.SummarizeMessages(messages)
	if err != nil {
		return err
	}

	return s.summaryRepo.SaveSummary(&Summary{
		ChannelID: channelID,
		Summary:   summary,
		CreatedAt: time.Now(),
	})
}
