package main

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"mempool/pkg/constants"
	"mempool/pkg/logging"
	"mempool/pkg/types"
)

func main() {
	godotenv.Load(".env")
	maxMempoolSize := os.Getenv(constants.ENV_MAX_MEMPOOL_SIZE)
	logger, err := logging.Logger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	logger.Sugar().Infof("initializing mempool of size [%v]", maxMempoolSize)
	if maxPoolSize, err := strconv.Atoi(maxMempoolSize); err != nil {
		logger.Sugar().Fatalf("[%s] envoirnment variable not set", constants.ENV_MAX_MEMPOOL_SIZE)
	} else {
		numOfCores := uint8(runtime.NumCPU())
		mempool, err := types.NewMempool(uint32(maxPoolSize), logger)
		if err != nil {
			logger.Sugar().Fatalf("error initializing mempool: [%v]", err)
		}
		waitGroup := &sync.WaitGroup{}
		mempool.StartProcessors(waitGroup, numOfCores) // Start processors equal to the number of CPU cores for CPU-bound tasks
		logger.Sugar().Infof("[%v] workers started", numOfCores)
		// start timer to test performance
		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			logger.Sugar().Infof("Total time taken to process transactions: %s", elapsed)
		}()
		// Process transactions.txt and insert into mempool
		logger.Sugar().Info("retrieving transactions and inserting into mempool")
		if transactionFile, err := os.Open(os.Getenv(constants.ENV_TRANSACTIONS_FILE_PATH)); err != nil {
			logger.Sugar().Errorf("error opening transactions.txt. ensure [%s] enviornment variable is set", constants.ENV_TRANSACTIONS_FILE_PATH)
		} else {
			defer transactionFile.Close()
			scanner := bufio.NewScanner(transactionFile)
			scanner.Split(bufio.ScanLines)
			var currentLine int
			for scanner.Scan() {
				currentLine++
				rawTransaction := strings.Fields(scanner.Text())
				if len(rawTransaction) != 4 {
					logger.Sugar().Errorf("transaction file at path [%s] is misformatted at line [%v]", constants.ENV_TRANSACTIONS_FILE_PATH, currentLine)
					continue
				}
				txHash := strings.TrimPrefix(rawTransaction[0], "TxHash=")
				gas, err := strconv.ParseFloat(strings.TrimPrefix(rawTransaction[1], "Gas="), 64)
				if err != nil {
					logger.Sugar().Errorf("gas conversion error for transaction with hash [%s] on line [%v]", txHash, currentLine)
					continue
				}
				feePerGas, err := strconv.ParseFloat(strings.TrimPrefix(rawTransaction[2], "FeePerGas="), 64)
				if err != nil {
					logger.Sugar().Errorf("feePerGas conversion error for transaction with hash [%s] on line [%v]", txHash, currentLine)
					continue
				}
				signature := strings.TrimPrefix(rawTransaction[3], "Signature=")
				err = mempool.AddTx(types.NewTx(logger, txHash, signature, gas, feePerGas), waitGroup)
				if err != nil {
					logger.Sugar().Errorf("error inserting transaction with hash [%s]: [%v]", txHash, err.Error())
					continue
				}
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
