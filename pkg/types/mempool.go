package types

import (
	"container/heap"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"

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
	AddTx(tx *Tx, group *sync.WaitGroup) (err error)       // Adds a transaction to the mempool, processing it in a goroutine.
	GetTx(txHash string) (*Tx, bool)                       // Retrieves a transaction by its hash from the mempool.
	MempoolLen() int                                       // Returns the current number of transactions in the mempool.
	CloseTxInsertChan()                                    // Closes the transaction insertion channel.
	ExportToFile() error                                   // Exports the mempool contents to a file.
	MaxMemPoolSize() uint32                                // Returns the maximum size of the mempool.
	StartProcessors(wg *sync.WaitGroup, numProcessors int) // Starts a specified number of goroutines to process transactions from the mempool.
}

var _ Mempool = (*mempool)(nil)

func NewMempool(maxPoolSize uint32, ls logging.LoggingSystem) (Mempool, error) {
	if maxPoolSize <= 0 {
		return nil, ErrMempoolSize
	}
	return &mempool{
		mu:             &sync.Mutex{},
		maxMemPoolSize: maxPoolSize,
		logger:         ls,
		txMap:          make(map[string]*Tx, maxPoolSize),
		txHeap:         make(TxHeap, 0, maxPoolSize),
		txChan:         make(chan *Tx, 500000), // Consider if this buffer size is optimal
		// Initialize new fields
		muPendingChecks: &sync.Mutex{},
		pendingChecks:   make(map[string]struct{}),
	}, nil
}

func (mp *mempool) MaxMemPoolSize() uint32 {
	return mp.maxMemPoolSize
}

func (mp *mempool) AddTx(tx *Tx, group *sync.WaitGroup) (err error) {
	mp.logger.Sugar().Named("mempool/AddTx").Debugf("calculating total fee for transaction with hash [%s]", tx.TxHash)
	tx.calculateTotalFees()

	// Check 1: Is it already fully processed and in the main Transactions map?
	mp.mu.Lock()
	if _, exists := mp.txMap[tx.TxHash]; exists {
		mp.mu.Unlock()
		mp.logger.Sugar().Named("mempool/AddTx").Warnf("rejected duplicate transaction (already in main pool) with hash [%s]", tx.TxHash)
		return errors.Errorf("Transaction with hash [%s] already exists in mempool", tx.TxHash)
	}
	mp.mu.Unlock()

	// Check 2: Is it currently pending processing (in txChan or about to be)?
	mp.muPendingChecks.Lock()
	if _, pending := mp.pendingChecks[tx.TxHash]; pending {
		mp.muPendingChecks.Unlock()
		mp.logger.Sugar().Named("mempool/AddTx").Warnf("rejected duplicate transaction (pending processing) with hash [%s]", tx.TxHash)
		return errors.Errorf("Transaction with hash [%s] is already pending processing", tx.TxHash)
	}
	// If not pending, mark it as pending before sending to channel
	mp.pendingChecks[tx.TxHash] = struct{}{}
	mp.muPendingChecks.Unlock()

	// Only increment WaitGroup if the transaction will actually be sent to the channel
	group.Add(1)
	mp.txChan <- tx
	mp.logger.Sugar().Named("mempool/AddTx").Debugf("Transaction with hash [%s] accepted and sent to processing channel", tx.TxHash)
	return nil // Successfully queued
}

// StartProcessors starts a specified number of goroutines to process transactions from the mempool.
func (mp *mempool) StartProcessors(wg *sync.WaitGroup, numProcessors int) {
	for i := 0; i < numProcessors; i++ {
		go mp.processTx(wg, mp.txChan)
	}
}

// processTx processes transactions from the txReadOnly channel.
func (mp *mempool) processTx(wg *sync.WaitGroup, txReadOnly <-chan *Tx) {
	// This WaitGroup 'wg' is for the test to wait for all its submitted transactions
	// to complete processing. Each transaction processed will call Done().
	// Note: wg.Add(1) should be called by the sender or handled carefully here.
	// The original design had wg.Add(1) here.

	for transaction := range txReadOnly { // Loop until channel is closed
		currentTxHash := transaction.TxHash
		mp.logger.Sugar().Named("mempool/processTx").Debugf("Processing transaction with hash [%s]", currentTxHash)

		// Remove from pendingChecks now that we've picked it up for processing.
		mp.muPendingChecks.Lock()
		delete(mp.pendingChecks, currentTxHash)
		mp.muPendingChecks.Unlock()

		mp.mu.Lock() // Lock for main Transactions map operations

		// Final check for duplicates right before insertion attempt.
		if _, exists := mp.txMap[currentTxHash]; exists {
			mp.logger.Sugar().Named("mempool/processTx").Warnf("Transaction with hash [%s] already exists in main pool (caught by final processor check). Discarding.", currentTxHash)
			mp.mu.Unlock()
			wg.Done() // Signal completion for this transaction
			continue
		}

		// Logic for when mempool is full: prioritize transactions with higher fee
		if uint32(len(mp.txHeap)) >= mp.maxMemPoolSize {
			// Pool full: check if new tx has higher priority than min
			minTx := mp.txHeap[0]
			if minTx.TotalFee < transaction.TotalFee {
				// Drop min
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
	mp.logger.Sugar().Named("mempool/processTx").Info("Channel closed, processor shutting down.")
}

// ExportToFile exports the contents of the mempool to a file, sorted by TotalFee ascending.
func (mp *mempool) ExportToFile() error {
	var sb strings.Builder
	mp.logger.Sugar().Infof("Exporting %d transactions", len(mp.txMap))
	for mp.txHeap.Len() > 0 {
		tx := heap.Pop(&mp.txHeap).(*Tx)
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
	mp.logger.Sugar().Infof("Exported %d bytes to file %s", bytes, fileName)
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
func (mp *mempool) MempoolLen() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return len(mp.txMap)
}
