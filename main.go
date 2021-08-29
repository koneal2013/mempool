package main

import (
	"bufio"
	"github.com/joho/godotenv"
	"kava-challange/pkg/constants"
	"kava-challange/pkg/logging"
	"kava-challange/pkg/types"
	"os"
	"strconv"
	"strings"
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
		//Process transactions.txt and insert into mempool
		if transactionFile, err := os.Open(os.Getenv(constants.ENV_TRANSACTIONS_FILE_PATH)); err != nil {
			logger.Sugar().Errorf("error opening transactions.txt. ensure [%s] enviornment variable is set", constants.ENV_TRANSACTIONS_FILE_PATH)
		} else {
			defer transactionFile.Close()
			scanner := bufio.NewScanner(transactionFile)
			scanner.Split(bufio.ScanLines)
			currentLine := 0
			for scanner.Scan() {
				currentLine++
				rawTransaction := strings.Fields(scanner.Text())
				if len(rawTransaction) != 4 {
					logger.Sugar().Errorf("transaction file at path [%s] is misformatted at line [%v]", constants.ENV_TRANSACTIONS_FILE_PATH, currentLine)
					continue
				}
				txHash := strings.TrimPrefix(rawTransaction[0], "TxHash=")
				if gas, err := strconv.ParseFloat(strings.TrimPrefix(rawTransaction[1], "Gas="), 64); err != nil {
					logger.Sugar().Errorf("gas conversion error for transaction with hash [%s] on line [%v]", txHash, currentLine)
					continue
				} else if feePerGas, err := strconv.ParseFloat(strings.TrimPrefix(rawTransaction[2], "FeePerGas="), 64); err != nil {
					logger.Sugar().Errorf("feePerGas conversion error for transaction with hash [%s] on line [%v]", txHash, currentLine)
					continue
				} else {
					signature := strings.TrimPrefix(rawTransaction[3], "Signature=")
					if err = mempool.AddTx(types.NewTx(logger, txHash, signature, gas, feePerGas)); err != nil {
						logger.Sugar().Error(err)
					}
				}
			}
			//export mempool to "prioritized-transactions.txt"
			mempool.ExportToFile()
		}
	}
	logger.Sugar().Named("main").Info("Done...")
}
