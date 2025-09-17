package utils

import (
	"go.uber.org/zap"
)

func NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	
	return logger, nil
}

func NewDevelopmentLogger() (*zap.Logger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	
	return logger, nil
}