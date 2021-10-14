package types

import (
	"mempool/pkg/logging"
)

type Tx struct {
	TxHash    string
	Gas       float64
	FeePerGas float64
	TotalFee  float64
	Signature string
}

type TxI interface {
	calculateTotalFees()
}

const (
	WARN_BAD_DATA = "encountered one or more missing parameters while creating transaction"
)

func NewTx(logger logging.LoggingSystem, txHash, signature string, gas, feePerGas float64) *Tx {
	if txHash == " " || signature == " " || gas == 0.0 || feePerGas == 0.0 {
		logger.Warn(WARN_BAD_DATA)
	}
	return &Tx{
		TxHash:    txHash,
		Gas:       gas,
		FeePerGas: feePerGas,
		Signature: signature,
	}
}

func (tx *Tx) calculateTotalFees() {
	tx.TotalFee = tx.FeePerGas * tx.Gas
}
