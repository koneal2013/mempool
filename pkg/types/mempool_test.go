package types_test

import (
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mempool/mocks"
	"mempool/pkg/logging"
	"mempool/pkg/types"
)

func TestNewMempool(t *testing.T) {
	for _, tc := range []struct {
		name        string
		maxPoolSize uint32
		isError     bool
		isFatal     bool
	}{
		{
			name:        "success",
			maxPoolSize: 1,
			isError:     false,
		},
		{
			name:        "failure_error",
			maxPoolSize: 0,
			isError:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := mocks.NewMockLoggingSystem(ctrl)

			result, err := types.NewMempool(tc.maxPoolSize, mockLogger)
			if tc.isError {
				require.ErrorIs(t, err, types.ErrMempoolSize)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			if !tc.isError {
				assert.Equalf(t, tc.maxPoolSize, result.MaxMemPoolSize(), "MaxMemPoolSize should match the initial configuration")
			}
		})
	}
}

func TestMempool_AddTx(t *testing.T) {
	for _, tc := range []struct {
		name        string
		txHash      string
		gas         float64
		feePerGas   float64
		signature   string
		maxPoolSize uint32
	}{
		{
			name:        "success_add_and_replace_if_higher_prio_when_full",
			txHash:      "txHash_original_low_prio", // This will be the low priority tx
			gas:         54.5,
			feePerGas:   0.1, // Explicitly low priority
			signature:   "testSigLow",
			maxPoolSize: 1,
		},
		{
			name:        "handle_duplicate_tx_when_space_available", // Renamed from failure_duplicate_tx
			txHash:      "txHash_dup",
			gas:         54.5,
			feePerGas:   0.4934,
			signature:   "testSigDup",
			maxPoolSize: 2, // Enough space for the original and another distinct tx
		},
		{
			name:        "success_add_two_distinct_and_ignore_third_duplicate", // Renamed and clarified
			txHash:      "txHash_first_of_two",
			gas:         54.5,
			feePerGas:   0.4934,
			signature:   "testSigFirst",
			maxPoolSize: 2, // Space for two distinct transactions
		},
		{
			name:        "success_drop_lowest_on_overflow_and_add_higher",
			txHash:      "txHash_to_be_dropped_eventually", // This is the initial low priority tx
			gas:         50.0,
			feePerGas:   0.1, // Lowest priority
			signature:   "sig_dropped",
			maxPoolSize: 1, // Mempool will be full with 1 tx
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := logging.Logger()
			require.NoError(t, err, "Failed to initialize logger for test")
			memPool, err := types.NewMempool(tc.maxPoolSize, logger)
			require.NoError(t, err)

			// Transaction defined by the test case parameters
			txFromTestCase := types.NewTx(logger, tc.txHash, tc.signature, tc.gas, tc.feePerGas)

			// A standard high-priority transaction for various test cases
			txHighPriority := types.NewTx(logger, "txHash_highPriority_standard", "sigHP", 60.0, 2.0) // Higher feePerGas

			// Another distinct transaction, different from txFromTestCase and txHighPriority
			txAnotherDistinct := types.NewTx(logger, "txHash_another_distinct", "sigAD", 55.0, 0.5)

			wg := &sync.WaitGroup{}
			memPool.StartProcessors(wg, 2) // Start processors before adding transactions

			switch tc.name {
			case "success_add_and_replace_if_higher_prio_when_full":
				errAddLowPrio := memPool.AddTx(txFromTestCase, wg)
				// Attempt to add the same low-priority transaction again (should be rejected by AddTx)
				errAddDuplicateLowPrio := memPool.AddTx(txFromTestCase, wg)
				errAddHighPrio := memPool.AddTx(txHighPriority, wg)

				memPool.CloseTxInsertChan()
				wg.Wait()

				assert.Nil(t, errAddLowPrio, "Adding the initial low priority transaction should succeed")
				assert.Error(t, errAddDuplicateLowPrio, "Adding a duplicate low priority transaction should fail (caught by AddTx)")
				assert.Nil(t, errAddHighPrio, "Adding the high priority transaction should succeed (replacing low prio)")

				assert.Equal(t, 1, memPool.MempoolLen(), "Mempool length should be 1") // Use MempoolLen()
				finalTx, inPool := memPool.GetTx(txHighPriority.TxHash)                // Use GetTx()
				assert.True(t, inPool, "High priority transaction should be in the mempool")
				if inPool { // Added check for finalTx to avoid panic if not in pool
					require.NotNil(t, finalTx, "High priority transaction pointer should not be nil if in pool")
					assert.Equal(t, txHighPriority.TotalFee, finalTx.TotalFee) // Direct access to TotalFee as GetTx returns *Tx
				}
				_, inPoolOriginal := memPool.GetTx(txFromTestCase.TxHash) // Use GetTx()
				assert.False(t, inPoolOriginal, "Original low priority transaction should have been replaced")

			case "handle_duplicate_tx_when_space_available":
				errAddOriginal := memPool.AddTx(txFromTestCase, wg)
				// Attempt to add the same transaction again (should be rejected by AddTx)
				errAddDuplicate := memPool.AddTx(txFromTestCase, wg)
				// Add a different transaction (should succeed)
				errAddDistinct := memPool.AddTx(txHighPriority, wg)

				memPool.CloseTxInsertChan()
				wg.Wait()

				assert.Nil(t, errAddOriginal, "Adding the initial transaction should succeed")
				assert.Error(t, errAddDuplicate, "Adding a duplicate transaction should fail (caught by AddTx)")
				assert.Nil(t, errAddDistinct, "Adding the distinct transaction should succeed")

				assert.Equal(t, 2, memPool.MempoolLen(), "Mempool length should be 2") // Use MempoolLen()
				_, inPool := memPool.GetTx(txFromTestCase.TxHash)                      // Use GetTx()
				assert.True(t, inPool, "Original transaction should be in the mempool")
				_, inPoolHP := memPool.GetTx(txHighPriority.TxHash) // Use GetTx()
				assert.True(t, inPoolHP, "Distinct (high priority) transaction should be in the mempool")

			case "success_add_two_distinct_and_ignore_third_duplicate":
				errAddFirst := memPool.AddTx(txFromTestCase, wg)
				errAddSecondDistinct := memPool.AddTx(txAnotherDistinct, wg)
				// Attempt to add a duplicate of the first transaction (should be rejected by AddTx)
				errAddDuplicateOfFirst := memPool.AddTx(txFromTestCase, wg)

				memPool.CloseTxInsertChan()
				wg.Wait()

				assert.Nil(t, errAddFirst, "Adding the first transaction should succeed")
				assert.Nil(t, errAddSecondDistinct, "Adding the second distinct transaction should succeed")
				assert.Error(t, errAddDuplicateOfFirst, "Adding a duplicate of the first transaction should fail (caught by AddTx)")

				assert.Equal(t, 2, memPool.MempoolLen(), "Mempool length should be 2") // Use MempoolLen()
				_, inPoolFirst := memPool.GetTx(txFromTestCase.TxHash)                 // Use GetTx()
				assert.True(t, inPoolFirst, "First transaction should be in the mempool")
				_, inPoolSecond := memPool.GetTx(txAnotherDistinct.TxHash) // Use GetTx()
				assert.True(t, inPoolSecond, "Second distinct transaction should be in the mempool")

			case "success_drop_lowest_on_overflow_and_add_higher":
				errAddLowPrio := memPool.AddTx(txFromTestCase, wg)
				// Attempt to add a duplicate of the low priority (should fail by AddTx)
				errAddDuplicateLowPrio := memPool.AddTx(txFromTestCase, wg)
				errAddHighPrioToReplace := memPool.AddTx(txHighPriority, wg)

				memPool.CloseTxInsertChan()
				wg.Wait()

				assert.Nil(t, errAddLowPrio, "Adding the low priority transaction should succeed")
				assert.Error(t, errAddDuplicateLowPrio, "Adding a duplicate of the low priority tx should fail (caught by AddTx)")
				assert.Nil(t, errAddHighPrioToReplace, "Adding the high priority transaction (to cause replacement) should succeed")

				assert.Equal(t, 1, memPool.MempoolLen(), "Mempool length should be 1 after replacement") // Use MempoolLen()

				finalTx, inPool := memPool.GetTx(txHighPriority.TxHash) // Use GetTx()
				assert.True(t, inPool, "High priority transaction should be in the mempool after replacement")
				if inPool { // Added check for finalTx to avoid panic if not in pool
					require.NotNil(t, finalTx, "High priority transaction pointer should not be nil if in pool")
					assert.Equal(t, txHighPriority.TotalFee, finalTx.TotalFee) // Direct access to TotalFee as GetTx returns *Tx
				}
				_, inPoolOriginal := memPool.GetTx(txFromTestCase.TxHash) // Use GetTx()
				assert.False(t, inPoolOriginal, "Original low priority transaction should have been dropped/replaced")
			}
		})
	}
}

func TestMempool_ExportToFile(t *testing.T) {
	for _, tc := range []struct {
		name        string
		txHash      string
		gas         float64
		feePerGas   float64
		signature   string
		isError     bool
		maxPoolSize uint32
	}{
		{
			name:        "success",
			txHash:      "txHash",
			gas:         54.5,
			feePerGas:   0.38483,
			signature:   "testSig",
			isError:     false,
			maxPoolSize: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			logger, err := logging.Logger()
			require.NoError(t, err, "Failed to initialize logger for test")
			memPool, err := types.NewMempool(tc.maxPoolSize, logger)
			require.NoError(t, err)
			tx := types.NewTx(logger, tc.txHash, tc.signature, tc.gas, tc.feePerGas)
			wg := &sync.WaitGroup{}

			err = memPool.AddTx(tx, wg)
			require.NoError(t, err)
			err = memPool.ExportToFile()
			require.NoError(t, err)

			if !tc.isError {
				assert.Nil(t, err)
				assert.FileExists(t, "./prioritized-transactions.txt")
			}
		})
	}
}

func BenchmarkMempool_AddTx(b *testing.B) {
	logger, err := logging.Logger()
	require.NoError(b, err, "Failed to initialize logger for benchmark")
	sizes := []int{100, 1000, 10000, 100000} // Different numbers of transactions to add

	for _, numTxs := range sizes {
		b.Run(fmt.Sprintf("NumTxs-%d", numTxs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()                                            // Stop timer for setup
				memPool, err := types.NewMempool(uint32(numTxs), logger) // Max pool size same as numTxs for this benchmark
				require.NoError(b, err)
				txs := make([]*types.Tx, numTxs)
				for j := 0; j < numTxs; j++ {
					txs[j] = generateUniqueTx(logger, j) // Generate a unique transaction
				}
				wg := &sync.WaitGroup{}
				b.StartTimer() // Restart timer for the actual operation

				for j := 0; j < numTxs; j++ {
					memPool.AddTx(txs[j], wg)
				}
				memPool.CloseTxInsertChan()
				wg.Wait() // Wait for all processing goroutines to finish their current tasks

				b.StopTimer() // Stop timer after operation
			}
		})
	}
}

func BenchmarkMempool_ExportToFile(b *testing.B) {
	logger, err := logging.Logger()
	require.NoError(b, err, "Failed to initialize logger for benchmark")
	sizes := []int{100, 1000, 10000, 50000} // Different mempool sizes to export

	for _, size := range sizes {
		b.Run(fmt.Sprintf("PoolSize-%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer() // Stop timer for setup
				memPool, err := types.NewMempool(uint32(size), logger)
				require.NoError(b, err)
				wg := &sync.WaitGroup{}
				for j := 0; j < size; j++ {
					tx := generateUniqueTx(logger, j) // Generate a unique transaction
					memPool.AddTx(tx, wg)
				}
				memPool.CloseTxInsertChan()
				wg.Wait() // Ensure all transactions are processed and inserted

				b.StartTimer() // Restart timer for the actual operation
				err = memPool.ExportToFile()
				if err != nil {
					b.Fatalf("ExportToFile failed: %v", err)
				}
				b.StopTimer() // Stop timer after operation

				// Clean up the created file
				os.Remove("./prioritized-transactions.txt")
			}
		})
	}
}

// Helper function to generate a unique transaction for benchmarks
func generateUniqueTx(logger logging.LoggingSystem, id int) *types.Tx {
	return types.NewTx(logger, fmt.Sprintf("txHash-%d-%d", id, time.Now().UnixNano()), "signature", rand.Float64()*100, rand.Float64()*10)
}
