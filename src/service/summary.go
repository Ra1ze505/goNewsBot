package service

import (
	"context"
	"fmt"
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
	for peerID, _ := range config.Channels {
		if err := s.ProcessChannelSummaries(peerID); err != nil {
			log.Errorf("Error processing summary for channel with peer_id %d: %v", peerID, err)
			continue
		}
	}
	log.Info("Successfully processed summaries for all channels")
	return nil
}

func (s *SummaryService) ProcessChannelSummaries(peerID int64) error {
	// Check if we already have a summary for today
	hasSummary, err := s.summaryRepo.HasSummaryToday(peerID)
	if err != nil {
		return fmt.Errorf("failed to check summary existence for channel %d: %w", peerID, err)
	}
	if hasSummary {
		log.Infof("Summary already exists for channel %d today", peerID)
		return nil
	}

	// Get messages for the channel
	messages, err := s.summaryRepo.GetMessagesForLastDay(peerID)
	if err != nil {
		return fmt.Errorf("failed to get messages for channel %d: %w", peerID, err)
	}

	if len(messages) == 0 {
		log.Infof("No messages found for channel %d", peerID)
		return nil
	}

	// Process messages and create summary
	summary, err := s.mlRepo.SummarizeMessages(messages)
	if err != nil {
		return fmt.Errorf("failed to generate summary for channel %d: %w", peerID, err)
	}

	// Save summary
	if err := s.summaryRepo.SaveSummary(&repository.Summary{
		ChannelID: peerID,
		Summary:   summary,
		CreatedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to save summary for channel %d: %w", peerID, err)
	}

	log.Infof("Successfully processed summary for channel %d", peerID)
	return nil
}
