package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Warn("Error loading .env file")
	}
}

func main() {
	log.Info("Starting historical summary generator...")
	loadEnv()

	// Connect to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to open database connection"))
	}
	defer db.Close()

	// Initialize repositories
	summaryRepo := repository.NewSummaryRepository(db)
	mlRepo, err := repository.NewMLRepository()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to initialize ML repository"))
	}

	// Generate summaries for the last 5 days
	if err := generateHistoricalSummaries(summaryRepo, mlRepo); err != nil {
		log.Fatal(errors.Wrap(err, "Failed to generate historical summaries"))
	}

	log.Info("Historical summary generation completed successfully")
}

func generateHistoricalSummaries(summaryRepo repository.SummaryRepositoryInterface, mlRepo repository.MLRepositoryInterface) error {
	// Get current time in UTC
	now := time.Now().UTC()

	// Process each channel
	for peerID, channelName := range config.Channels {
		log.Infof("Processing channel: %s (ID: %d)", channelName, peerID)

		// Process each day for the last 5 days
		for i := 4; i >= 0; i-- {
			targetDate := now.AddDate(0, 0, -i)
			log.Infof("Processing date: %s", targetDate.Format("2006-01-02"))

			// Get messages for the specific day
			messages, err := summaryRepo.GetMessagesForDate(peerID, targetDate)
			if err != nil {
				log.Errorf("Error getting messages for date %s: %v", targetDate.Format("2006-01-02"), err)
				continue
			}

			if len(messages) == 0 {
				log.Infof("No messages found for channel %d on %s", peerID, targetDate.Format("2006-01-02"))
				continue
			}

			log.Infof("Found %d messages for summarization", len(messages))

			// Generate summary
			summary, err := mlRepo.SummarizeMessages(messages)
			if err != nil {
				log.Errorf("Error generating summary: %v", err)
				continue
			}

			// Print summary to console
			fmt.Printf("\n=== Summary for %s (%s) ===\n", channelName, targetDate.Format("2006-01-02"))
			fmt.Println(summary)
			fmt.Println("================================")
		}
	}

	return nil
}
