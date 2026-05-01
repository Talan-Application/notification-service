package logger

import "go.uber.org/zap"

func New(env string) *zap.Logger {
	var log *zap.Logger

	if env == "production" {
		log, _ = zap.NewProduction()
	} else {
		log, _ = zap.NewDevelopment()
	}

	return log
}
