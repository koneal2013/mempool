package models

type Transaction struct {
	TxHash    string
	Gas       int
	FeePerGas float64
	Signature string
}
