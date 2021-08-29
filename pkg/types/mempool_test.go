package types_test

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"kava-challange/mocks"
	"kava-challange/pkg/logging"
	"kava-challange/pkg/types"
	"testing"
)

func TestNewMempool(t *testing.T) {
	for _, tc := range []struct {
		name        string
		maxPoolSize int
		isError     bool
		isFatal     bool
	}{
		{
			name:        "success",
			maxPoolSize: 1,
			isError:     false,
			isFatal:     false,
		},
		{
			name:        "failure_fatal",
			maxPoolSize: 0,
			isError:     false,
			isFatal:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := mocks.NewMockLoggingSystem(ctrl)

			if tc.isFatal {
				mockLogger.EXPECT().Fatal(types.ERR_MEMPOOL_SIZE)
			}

			result := types.NewMempool(tc.maxPoolSize, mockLogger)

			if !tc.isError {
				assert.NotNil(t, result.Transactions)
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
		isError     bool
		maxPoolSize int
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
		{
			name:        "failure_duplicate_tx",
			txHash:      "txHash_dup",
			gas:         54.5,
			feePerGas:   0.4934,
			signature:   "testSig",
			isError:     true,
			maxPoolSize: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			logger := logging.Logger()
			memPool := types.NewMempool(tc.maxPoolSize, logger)
			tx := types.NewTx(logger, tc.txHash, tc.signature, tc.gas, tc.feePerGas)

			original := memPool.AddTx(tx)
			duplicate := memPool.AddTx(tx)

			if tc.isError {
				assert.Error(t, duplicate)
			} else {
				assert.Nil(t, original)
				assert.Equal(t, len(memPool.Transactions), 1)
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
		maxPoolSize int
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
			logger := logging.Logger()
			memPool := types.NewMempool(tc.maxPoolSize, logger)
			tx := types.NewTx(logger, tc.txHash, tc.signature, tc.gas, tc.feePerGas)

			err := memPool.AddTx(tx)
			err = memPool.ExportToFile()

			if !tc.isError {
				assert.Nil(t, err)
				assert.FileExists(t, "./prioritized-transactions.txt")
			}
		})
	}
}
