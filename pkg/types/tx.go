package types

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

func NewTx(txHash, signature string, gas, feePerGas float64) *Tx {
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
