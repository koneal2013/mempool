package main

import (
	"github.com/joho/godotenv"
	"kava-challange/pkg/constants"
	"kava-challange/pkg/logging"
	"kava-challange/pkg/types"
	"os"
	"strconv"
)

func main() {
	godotenv.Load(".env")
	maxMempoolSize := os.Getenv(constants.ENV_MAX_MEMPOOL_SIZE)
	logger := logging.Logger()
	defer logger.Sync()
	logger.Sugar().Infof("initializing mempool of size [%v]", maxMempoolSize)
	if maxPoolSize, err := strconv.Atoi(maxMempoolSize); err != nil {
		logger.Sugar().Fatalf("[%s] envoirnment variable not set", constants.ENV_MAX_MEMPOOL_SIZE)
	} else {
		logger.Sugar().Info("retrieving transactions and inserting into mempool")
		mempool := types.NewMempool(maxPoolSize, logger)
		mempool.AddTx(types.NewTx("hash", "sig", 542.55, 1.01))

	}
	logger.Sugar().Named("main").Info("Done...")
}
