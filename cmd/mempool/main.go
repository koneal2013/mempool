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
	"go.uber.org/zap"

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
	logger.Info("initializing mempool", zap.String("maxMempoolSize", maxMempoolSize))
	if maxPoolSize, err := strconv.Atoi(maxMempoolSize); err != nil {
		logger.Fatal("environment variable not set", zap.String("variable", constants.ENV_MAX_MEMPOOL_SIZE))
	} else {
		numOfCores := uint8(runtime.NumCPU())
		mempool, err := types.NewMempool(uint32(maxPoolSize), logger)
		if err != nil {
			logger.Fatal("error initializing mempool", zap.Error(err))
		}
		waitGroup := &sync.WaitGroup{}
		mempool.StartProcessors(waitGroup, numOfCores) // Start processors equal to the number of CPU cores for CPU-bound tasks
		logger.Info("workers started", zap.Uint8("count", numOfCores))
		// start timer to test performance
		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			logger.Info("Total time taken to process transactions", zap.Duration("duration", elapsed))
		}()
		// Process transactions.txt and insert into mempool
		logger.Info("retrieving transactions and inserting into mempool")
		if transactionFile, err := os.Open(os.Getenv(constants.ENV_TRANSACTIONS_FILE_PATH)); err != nil {
			logger.Error("error opening transactions.txt. ensure environment variable is set", zap.String("variable", constants.ENV_TRANSACTIONS_FILE_PATH))
		} else {
			defer transactionFile.Close()
			scanner := bufio.NewScanner(transactionFile)
			scanner.Split(bufio.ScanLines)
			var currentLine uint32
			for scanner.Scan() {
				currentLine++
				rawTransaction := strings.Fields(scanner.Text())
				if len(rawTransaction) != 4 {
					logger.Error("transaction file is misformatted", zap.String("path", constants.ENV_TRANSACTIONS_FILE_PATH), zap.Uint32("line", currentLine))
					continue
				}
				txHash := strings.TrimPrefix(rawTransaction[0], "TxHash=")
				gas, err := strconv.ParseFloat(strings.TrimPrefix(rawTransaction[1], "Gas="), 64)
				if err != nil {
					logger.Error("gas conversion error", zap.String("txHash", txHash), zap.Uint32("line", currentLine))
					continue
				}
				feePerGas, err := strconv.ParseFloat(strings.TrimPrefix(rawTransaction[2], "FeePerGas="), 64)
				if err != nil {
					logger.Error("feePerGas conversion error", zap.String("txHash", txHash), zap.Uint32("line", currentLine))
					continue
				}
				signature := strings.TrimPrefix(rawTransaction[3], "Signature=")
				err = mempool.AddTx(types.NewTx(logger, txHash, signature, gas, feePerGas), waitGroup)
				if err != nil {
					logger.Error("error inserting transaction", zap.String("txHash", txHash), zap.Error(err))
					continue
				}
			}
			waitGroup.Wait()
			mempool.CloseTxInsertChan()
			if err = mempool.ExportToFile(); err != nil {
				logger.Error("error creating prioritized-transactions.txt", zap.Error(err))
			}
		}
	}
	logger.Named("main").Info("Done...")
}
