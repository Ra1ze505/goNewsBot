package service

import (
	"context"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
)

type SummaryService struct {
	summaryRepo repository.SummaryRepositoryInterface
	mlRepo      repository.MLRepositoryInterface
	// Channel to receive signals from message service
	messagesFetched <-chan struct{}
}

func NewSummaryService(summaryRepo repository.SummaryRepositoryInterface, mlRepo repository.MLRepositoryInterface, messagesFetched <-chan struct{}) *SummaryService {
	return &SummaryService{
		summaryRepo:     summaryRepo,
		mlRepo:          mlRepo,
		messagesFetched: messagesFetched,
	}
}

func (s *SummaryService) StartSummaryFetcher(ctx context.Context) {
	go s.startSummaryFetcher(ctx)
}

func (s *SummaryService) startSummaryFetcher(ctx context.Context) {
	if err := s.processAllChannels(); err != nil {
		log.Errorf("Error processing summaries on startup: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.messagesFetched:
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

	summary, err := s.mlRepo.SummarizeMessages(messages)
	if err != nil {
		return err
	}

	return s.summaryRepo.SaveSummary(&repository.Summary{
		ChannelID: channelID,
		Summary:   summary,
		CreatedAt: time.Now(),
	})
}
