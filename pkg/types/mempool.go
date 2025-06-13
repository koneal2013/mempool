package types

import (
	"container/heap"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"mempool/pkg/constants"
	"mempool/pkg/logging"
)

var (
	ErrMempoolSize = errors.New("mempool size cannot be less than or equal to 0")
)

type mempool struct {
	mu             *sync.Mutex    // Protects txMap and txHeap
	txMap          map[string]*Tx // O(1) lookup by hash
	txHeap         TxHeap         // Min-heap for priority management O(log n) for insertion and removal
	txChan         chan *Tx
	maxMemPoolSize uint32 // Maximum size of the mempool (max value of uint32 is 4,294,967,295)
	logger         logging.LoggingSystem

	// New fields for handling in-flight/pending transactions
	muPendingChecks *sync.Mutex
	pendingChecks   map[string]struct{} // Tracks hashes submitted to txChan but not yet in Transactions
}

type Mempool interface {
	AddTx(tx *Tx, group *sync.WaitGroup) (err error)         // Adds a transaction to the mempool, processing it in a goroutine.
	GetTx(txHash string) (*Tx, bool)                         // Retrieves a transaction by its hash from the mempool.
	MempoolLen() uint32                                      // Returns the current number of transactions in the mempool.
	CloseTxInsertChan()                                      // Closes the transaction insertion channel.
	ExportToFile() error                                     // Exports the mempool contents to a file.
	MaxMemPoolSize() uint32                                  // Returns the maximum size of the mempool.
	StartProcessors(wg *sync.WaitGroup, numProcessors uint8) // Starts a specified number of goroutines to process transactions from the mempool.
}

var _ Mempool = (*mempool)(nil)

func NewMempool(maxPoolSize uint32, ls logging.LoggingSystem) (Mempool, error) {
	if maxPoolSize <= 0 {
		return nil, ErrMempoolSize
	}
	return &mempool{
		mu:              &sync.Mutex{},
		maxMemPoolSize:  maxPoolSize,
		logger:          ls,
		txMap:           make(map[string]*Tx, maxPoolSize),
		txHeap:          make(TxHeap, 0, maxPoolSize),
		txChan:          make(chan *Tx, 200000), // Buffered channel to hold transactions before processing
		muPendingChecks: &sync.Mutex{},
		pendingChecks:   make(map[string]struct{}),
	}, nil
}

func (mp *mempool) MaxMemPoolSize() uint32 {
	return mp.maxMemPoolSize
}

func (mp *mempool) AddTx(tx *Tx, group *sync.WaitGroup) (err error) {
	mp.logger.Named("mempool/AddTx").Debug("calculating total fee for transaction", zap.String("txHash", tx.TxHash))
	tx.calculateTotalFees()

	// Check 1: Is it already fully processed and in the main Transactions map?
	mp.mu.Lock()
	if _, exists := mp.txMap[tx.TxHash]; exists {
		mp.mu.Unlock()
		mp.logger.Named("mempool/AddTx").Warn("rejected duplicate transaction (already in main pool)", zap.String("txHash", tx.TxHash))
		return errors.Errorf("Transaction with hash [%s] already exists in mempool", tx.TxHash)
	}
	mp.mu.Unlock()

	// Check 2: Is it currently pending processing (in txChan or about to be)?
	mp.muPendingChecks.Lock()
	if _, pending := mp.pendingChecks[tx.TxHash]; pending {
		mp.muPendingChecks.Unlock()
		mp.logger.Named("mempool/AddTx").Warn("rejected duplicate transaction (pending processing)", zap.String("txHash", tx.TxHash))
		return errors.Errorf("Transaction with hash [%s] is already pending processing", tx.TxHash)
	}
	// If not pending, mark it as pending before sending to channel
	mp.pendingChecks[tx.TxHash] = struct{}{}
	mp.muPendingChecks.Unlock()

	// Only increment WaitGroup if the transaction will actually be sent to the channel
	group.Add(1)
	mp.txChan <- tx
	mp.logger.Named("mempool/AddTx").Debug("Transaction with hash accepted and sent to processing channel", zap.String("txHash", tx.TxHash))
	return nil // Successfully queued
}

// StartProcessors starts a specified number of goroutines to process transactions from the mempool.
func (mp *mempool) StartProcessors(wg *sync.WaitGroup, numProcessors uint8) {
	for i := uint8(0); i < numProcessors; i++ {
		go mp.processTx(wg, mp.txChan)
	}
}

// processTx processes transactions from the txReadOnly channel.
func (mp *mempool) processTx(wg *sync.WaitGroup, txReadOnly <-chan *Tx) {
	for transaction := range txReadOnly { // Loop until channel is closed
		currentTxHash := transaction.TxHash
		mp.logger.Named("mempool/processTx").Debug("Processing transaction", zap.String("txHash", currentTxHash))

		// Remove from pendingChecks now that we've picked it up for processing.
		mp.muPendingChecks.Lock()
		delete(mp.pendingChecks, currentTxHash)
		mp.muPendingChecks.Unlock()

		mp.mu.Lock() // Lock for main Transactions map operations

		// Final check for duplicates right before insertion attempt.
		if _, exists := mp.txMap[currentTxHash]; exists {
			mp.logger.Named("mempool/processTx").Warn("Transaction already exists in main pool (caught by final processor check). Discarding.", zap.String("txHash", currentTxHash))
			mp.mu.Unlock()
			wg.Done() // Signal completion for this transaction
			continue
		}

		// Logic for when mempool is full: prioritize transactions with higher fee
		if uint32(len(mp.txHeap)) >= mp.maxMemPoolSize {
			// Pool full: check if new tx has higher priority than the current min (top of min-heap)
			minTx := mp.txHeap[0]
			if transaction.TotalFee > minTx.TotalFee {
				// Replace minTx with the new higher-fee transaction
				delete(mp.txMap, minTx.TxHash)
				heap.Pop(&mp.txHeap)
			} else {
				mp.mu.Unlock()
				wg.Done() // Signal completion for this transaction
				continue
			}
		}
		// Insert new tx
		heap.Push(&mp.txHeap, transaction)
		mp.txMap[currentTxHash] = transaction
		mp.mu.Unlock()
		wg.Done() // Signal completion for this transaction
	}
	mp.logger.Named("mempool/processTx").Info("Channel closed, processor shutting down.")
}

// ExportToFile exports the contents of the mempool to a file, sorted by TotalFee descending.
func (mp *mempool) ExportToFile() error {
	var sb strings.Builder
	txsDesc := make([]*Tx, 0, len(mp.txHeap))
	mp.logger.Info("Exporting transactions", zap.Int("count", len(mp.txHeap)))
	for mp.txHeap.Len() > 0 {
		tx := heap.Pop(&mp.txHeap).(*Tx)
		txsDesc = append(txsDesc, tx)
	}

	// Sort transactions by TotalFee in descending order
	for i := len(txsDesc) - 1; i >= 0; i-- {
		tx := txsDesc[i]
		fmt.Fprintf(&sb, "TxHash=%v Gas=%v FeePerGas=%v Signature=%v TotalFee=%v \n", tx.TxHash, tx.Gas, tx.FeePerGas, tx.Signature, tx.TotalFee)
	}

	fileName := os.Getenv(constants.PRIORITIZED_TX_FILE_PATH)
	if fileName == "" {
		fileName = "./prioritized-transactions.txt"
	}
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", fileName)
	}
	defer file.Close()
	bytes, err := file.WriteString(sb.String())
	if err != nil {
		return errors.Wrapf(err, "failed to write to file %s", fileName)
	}
	mp.logger.Info("Exported bytes to file", zap.Int("bytes", bytes), zap.String("fileName", fileName))
	return nil
}

// CloseTxInsertChan closes the transaction insertion channel.
func (mp *mempool) CloseTxInsertChan() {
	close(mp.txChan)
}

// GetTx retrieves a transaction from the mempool in a thread-safe manner.
func (mp *mempool) GetTx(txHash string) (*Tx, bool) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	tx, exists := mp.txMap[txHash]
	return tx, exists
}

// MempoolLen returns the current number of transactions in the mempool in a thread-safe manner.
func (mp *mempool) MempoolLen() uint32 {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return uint32(len(mp.txMap))
}
