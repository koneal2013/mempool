package main

import (
	"kava-challange/pkg/logging"
)

func main() {
	logger := logging.Logger()
	defer logger.Sync()
	logger.Sugar().Named("main").Info("Done...")
}
