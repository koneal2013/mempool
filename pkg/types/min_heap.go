package types

// TxHeap implements heap.Interface for *Tx based on TotalFee (min-heap)
type TxHeap []*Tx

func (h TxHeap) Len() int { return len(h) }

// Min-heap: Less returns true if i's TotalFee is less than j's
func (h TxHeap) Less(i, j int) bool { return h[i].TotalFee < h[j].TotalFee }
func (h TxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TxHeap) Push(x interface{}) {
	*h = append(*h, x.(*Tx))
}

func (h *TxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
