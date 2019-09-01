package main

import (
	"github.com/worldhistorymap/backend/pkg/scrapper"
	"github.com/worldhistorymap/backend/pkg/shared"
	"go.uber.org/zap"
)

func main() {
	logger, err := shared.GetLogger()
	defer logger.Sync()
	if err != nil {
		logger.Fatal("Error creating logger", zap.Error(err))
		return
	}
	config, err := shared.GetConfig()
	if err != nil {
		logger.Fatal("Error getting config", zap.Error(err))
		return
	}

	logger.Info("Running Scrapper")
	err = scrapper.NewScrapper(config, logger)
	if err != nil {
		logger.Fatal("Error running scrapper", zap.Error(err))
	}
	logger.Info("Scrapper Finished")
}
