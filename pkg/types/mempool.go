package types

import (
	"fmt"
	"github.com/pkg/errors"
	"kava-challange/pkg/logging"
	"sort"
)

var (
	maxMempoolSize int
	logger         logging.LoggingSystem
)

type mempool struct {
	Transactions []*Tx
}

type MempoolI interface {
	AddTx(tx *Tx) (err error)
	dropTx(tx *Tx) (err error)
	sort()
	contains(hash string) bool
}

func NewMempool(maxPoolSize int, ls logging.LoggingSystem) *mempool {
	maxMempoolSize = maxPoolSize
	logger = ls
	return &mempool{
		Transactions: make([]*Tx, 0),
	}
}

func (mp *mempool) AddTx(tx *Tx) (err error) {
	if exists := mp.contains(tx.TxHash); exists {
		err = fmt.Errorf("transaction with hash [%s] already exists", tx.TxHash)
		return err
	}
	logger.Sugar().Named("mempool/AddTx").Infof("calculating total fee for transaction with hash [%s]", tx.TxHash)
	tx.calculateTotalFees()
	//when mempool is full, prioritize transactions with higher fee
	if len(mp.Transactions) >= maxMempoolSize {
		if err = mp.dropTx(mp.Transactions[maxMempoolSize-1]); err != nil {
			return errors.Wrapf(err, "unable to add transaction with hash [%s] because the mempool is full", tx.TxHash)
		}
	}
	mp.Transactions = append(mp.Transactions, tx)
	mp.sort()
	return nil
}

func (mp *mempool) dropTx(tx *Tx) (err error) {
	if exists := mp.contains(tx.TxHash); exists {
		logger.Sugar().Named("mempool/dropTx").Infof("dropping low priority transaction with hash [%s] and total fee of [%v]", tx.TxHash, tx.TotalFee)
		mp.Transactions = mp.Transactions[:maxMempoolSize-1]
		return nil
	}
	err = fmt.Errorf("mempool does not contain a transaction with hash [%s]", tx.TxHash)
	return err
}

func (mp *mempool) sort() {
	sort.Slice(mp.Transactions, func(i, j int) bool {
		return mp.Transactions[i].TotalFee > mp.Transactions[j].TotalFee
	})
}

func (mp *mempool) contains(hash string) bool {
	for _, tx := range mp.Transactions {
		if mp.Transactions[0] != nil && tx.TxHash == hash {
			return true
		}
	}
	return false
}
