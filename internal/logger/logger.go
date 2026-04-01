package logger

import (
	"os"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func Log() *zap.SugaredLogger {
	return logger
}

func Flush() error {
	return logger.Sync()
}

func InitLogger() {
	appEnv := os.Getenv("APP_ENV")

	if appEnv != "DEV" {
		logger = zap.Must(zap.NewProduction()).Sugar()
	} else {
		logger = zap.Must(zap.NewDevelopment()).Sugar()
	}
}
