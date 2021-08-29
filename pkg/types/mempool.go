package types

import (
	"fmt"
	"github.com/pkg/errors"
	"kava-challange/pkg/logging"
	"os"
	"sort"
)

var (
	maxMempoolSize int
	logger         logging.LoggingSystem
)

const (
	ERR_MEMPOOL_SIZE = "mempool size cannot be less than or equal to 0"
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
	if maxPoolSize <= 0 {
		ls.Fatal(ERR_MEMPOOL_SIZE)
	}
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
	logger.Sugar().Named("mempool/AddTx").Debugf("calculating total fee for transaction with hash [%s]", tx.TxHash)
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
		logger.Sugar().Named("mempool/dropTx").Debugf("dropping low priority transaction with hash [%s] and total fee of [%v]", tx.TxHash, tx.TotalFee)
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
		if len(mp.Transactions) > 0 && tx.TxHash == hash {
			return true
		}
	}
	return false
}

func (mp *mempool) ExportToFile() (err error) {
	if prioritizedMempoolFile, err := os.Create("prioritized-transactions.txt"); err != nil {
		logger.Sugar().Named("mempool/ExportToFile").Error("unable to create file [prioritized-transactions.txt]")
		return err
	} else {
		defer prioritizedMempoolFile.Close()
		for _, tx := range mp.Transactions {
			if _, err = prioritizedMempoolFile.WriteString(fmt.Sprintf("TxHash=%v Gas=%v FeePerGas=%v Signature=%v \n", tx.TxHash, tx.Gas, tx.FeePerGas, tx.Signature)); err != nil {
				logger.Sugar().Named("mempool/ExportToFile").Errorf("unable to write [TxHash=%v Gas=%v FeePerGas=%v Signature=%v] to prioritized-transactions.txt", tx.TxHash, tx.Gas, tx.FeePerGas, tx.Signature)
				continue
			}
		}
	}
	return nil
}
