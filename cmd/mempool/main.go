package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"

	"mempool/pkg/constants"
	"mempool/pkg/logging"
	"mempool/pkg/types"
)

func main() {
	godotenv.Load(".env")
	maxMempoolSize := os.Getenv(constants.ENV_MAX_MEMPOOL_SIZE)
	logger := logging.Logger()
	defer logger.Sync()
	fmt.Println(len("hello")) // TODO: Remove this debug line
	logger.Sugar().Infof("initializing mempool of size [%v]", maxMempoolSize)
	if maxPoolSize, err := strconv.Atoi(maxMempoolSize); err != nil {
		logger.Sugar().Fatalf("[%s] envoirnment variable not set", constants.ENV_MAX_MEMPOOL_SIZE)
	} else {
		logger.Sugar().Info("retrieving transactions and inserting into mempool")
		mempool, err := types.NewMempool(maxPoolSize, logger) // Type will be inferred, no change needed here if methods are on interface
		if err != nil {
			logger.Sugar().Fatalf("error initializing mempool: [%v]", err)
		}
		// Process transactions.txt and insert into mempool
		if transactionFile, err := os.Open(os.Getenv(constants.ENV_TRANSACTIONS_FILE_PATH)); err != nil {
			logger.Sugar().Errorf("error opening transactions.txt. ensure [%s] enviornment variable is set", constants.ENV_TRANSACTIONS_FILE_PATH)
		} else {
			defer transactionFile.Close()
			scanner := bufio.NewScanner(transactionFile)
			scanner.Split(bufio.ScanLines)
			var currentLine int
			waitGroup := &sync.WaitGroup{}
			for scanner.Scan() {
				currentLine++
				rawTransaction := strings.Fields(scanner.Text())
				waitGroup.Add(1)
				go func(curLine int, rawTx []string, group *sync.WaitGroup) {
					defer group.Done()
					if len(rawTx) != 4 {
						logger.Sugar().Errorf("transaction file at path [%s] is misformatted at line [%v]", constants.ENV_TRANSACTIONS_FILE_PATH, currentLine)
						return
					}
					txHash := strings.TrimPrefix(rawTx[0], "TxHash=")
					if gas, err := strconv.ParseFloat(strings.TrimPrefix(rawTx[1], "Gas="), 64); err != nil {
						logger.Sugar().Errorf("gas conversion error for transaction with hash [%s] on line [%v]", txHash, curLine)
						return
					} else if feePerGas, err := strconv.ParseFloat(strings.TrimPrefix(rawTx[2], "FeePerGas="), 64); err != nil {
						logger.Sugar().Errorf("feePerGas conversion error for transaction with hash [%s] on line [%v]", txHash, curLine)
						return
					} else {
						signature := strings.TrimPrefix(rawTx[3], "Signature=")
						if err = mempool.AddTx(types.NewTx(logger, txHash, signature, gas, feePerGas), group); err != nil {
							logger.Sugar().Errorf("error inserting transaction with hash [%s]: [%v]", txHash, err.Error())
							return
						}
					}
				}(currentLine, rawTransaction, waitGroup)
			}
			waitGroup.Wait()
			mempool.CloseTxInsertChan()
			if err = mempool.ExportToFile(); err != nil {
				logger.Sugar().Error("error creating prioritized-transactions.txt", err)
			}
		}
	}
	logger.Sugar().Named("main").Info("Done...")
}
