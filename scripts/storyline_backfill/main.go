package main

import (
	"database/sql"
	"flag"
	"os"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/config"
	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/Ra1ze505/goNewsBot/src/service"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Warn("Error loading .env file")
	}
}

func main() {
	loadEnv()

	days := flag.Int("days", 7, "сколько прошлых дней прогнать")
	endDateStr := flag.String("end-date", "", "последний день replay в формате YYYY-MM-DD (по умолчанию вчера)")
	channelFlag := flag.Int64("channel", 0, "ID канала (по умолчанию все из config.Channels)")
	reset := flag.Bool("reset", true, "очистить storylines/observations канала перед прогоном")
	writeSummaries := flag.Bool("write-summaries", false, "писать ли дневной текст в summaries")
	flag.Parse()

	endDate := time.Now().UTC().AddDate(0, 0, -1)
	if *endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", *endDateStr)
		if err != nil {
			log.Fatal(errors.Wrap(err, "invalid --end-date"))
		}
		endDate = parsed.UTC()
	}
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to open database connection"))
	}
	defer db.Close()

	summaryRepo := repository.NewSummaryRepository(db)
	storylineRepo := repository.NewStorylineRepository(db)
	mlRepo, err := repository.NewMLRepository()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to initialize ML repository"))
	}
	processor := service.NewStorylineProcessor(summaryRepo, storylineRepo, mlRepo)

	channels := selectChannels(*channelFlag)

	for _, channelID := range channels {
		log.Infof("Backfilling channel %d (%s)", channelID, config.Channels[channelID])

		if *reset {
			if err := storylineRepo.ResetChannel(channelID); err != nil {
				log.Errorf("failed to reset channel %d: %v", channelID, err)
				continue
			}
			log.Infof("Reset storylines/observations for channel %d", channelID)
		}

		startDate := endDate.AddDate(0, 0, -(*days - 1))
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			processDay(processor, summaryRepo, channelID, d, *writeSummaries)
		}
	}

	log.Info("Storyline backfill completed")
}

func selectChannels(channelFlag int64) []int64 {
	if channelFlag != 0 {
		return []int64{channelFlag}
	}
	channels := make([]int64, 0, len(config.Channels))
	for id := range config.Channels {
		channels = append(channels, id)
	}
	return channels
}

func processDay(processor *service.StorylineProcessor, summaryRepo repository.SummaryRepositoryInterface, channelID int64, day time.Time, writeSummaries bool) {
	dayStr := day.Format("2006-01-02")

	msgs, err := summaryRepo.GetMessagesForDateWithIDs(channelID, day)
	if err != nil {
		log.Errorf("channel %d %s: failed to get messages: %v", channelID, dayStr, err)
		return
	}
	if len(msgs) == 0 {
		log.Infof("channel %d %s: no messages, skipping", channelID, dayStr)
		return
	}

	digest, err := processor.ProcessDay(channelID, day, msgs)
	if err != nil {
		log.Errorf("channel %d %s: ProcessDay failed: %v", channelID, dayStr, err)
		return
	}

	log.Infof("channel %d %s: processed %d messages", channelID, dayStr, len(msgs))

	if writeSummaries {
		if err := summaryRepo.SaveSummary(&repository.Summary{
			ChannelID: channelID,
			Summary:   digest,
			CreatedAt: day,
		}); err != nil {
			log.Errorf("channel %d %s: failed to save summary: %v", channelID, dayStr, err)
		}
	}
}
