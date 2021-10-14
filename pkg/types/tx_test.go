package types_test

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mempool/mocks"
	"mempool/pkg/types"
	"testing"
)

func TestNewTx(t *testing.T) {
	for _, tc := range []struct {
		name      string
		txHash    string
		gas       float64
		feePerGas float64
		signature string
		isWarning bool
	}{{
		name:      "success",
		txHash:    "testHash",
		gas:       0.254,
		feePerGas: 0.784,
		signature: "testSignature",
		isWarning: false,
	},
		{
			name:      "warning",
			txHash:    "",
			gas:       0.0,
			feePerGas: 0.0,
			signature: "",
			isWarning: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := mocks.NewMockLoggingSystem(ctrl)

			if tc.isWarning {
				mockLogger.EXPECT().Warn(types.WARN_BAD_DATA)
			}

			result := types.NewTx(mockLogger, tc.txHash, tc.signature, tc.gas, tc.feePerGas)

			if !tc.isWarning {
				assert.Equal(t, result.TxHash, tc.txHash)
				assert.Equal(t, result.Gas, tc.gas)
				assert.Equal(t, result.FeePerGas, tc.feePerGas)
				assert.Equal(t, result.Signature, tc.signature)
			}
		})
	}
}
