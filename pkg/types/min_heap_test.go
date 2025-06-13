package types

import (
	"container/heap"
	"testing"
)

func TestLenLessSwap(t *testing.T) {
	a := &Tx{TotalFee: 10}
	b := &Tx{TotalFee: 5}
	c := &Tx{TotalFee: 20}
	h := TxHeap{a, b, c}

	if h.Len() != 3 {
		t.Errorf("expected length 3, got %d", h.Len())
	}
	if !h.Less(1, 0) { // b < a
		t.Errorf("expected Less(1,0) to be true for min-heap")
	}
	h.Swap(0, 2)
	if h[0] != c || h[2] != a {
		t.Errorf("swap failed")
	}
}

func TestPushPop(t *testing.T) {
	h := &TxHeap{}
	heap.Init(h)
	heap.Push(h, &Tx{TotalFee: 15})
	heap.Push(h, &Tx{TotalFee: 5})
	heap.Push(h, &Tx{TotalFee: 25})
	heap.Push(h, &Tx{TotalFee: 10})

	expectedOrder := []int{5, 10, 15, 25}
	for _, want := range expectedOrder {
		got := heap.Pop(h).(*Tx).TotalFee
		if got != float64(want) {
			t.Errorf("expected %d, got %v", want, got)
		}
	}
}

func BenchmarkTxHeapPushPop(b *testing.B) {
	h := &TxHeap{}
	heap.Init(h)
	for i := 0; i < b.N; i++ {
		heap.Push(h, &Tx{TotalFee: float64(i)})
	}
	for h.Len() > 0 {
		heap.Pop(h)
	}
}
