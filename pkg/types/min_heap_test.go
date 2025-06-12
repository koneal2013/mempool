package types

import (
	"container/heap"
	"testing"
)

func TestLenLessSwap(t *testing.T) {
	a := &Tx{TotalFee: 10}
	b := &Tx{TotalFee: 20}
	c := &Tx{TotalFee: 5}

	h := TxHeap{a, b, c}

	// Test Len
	if got, want := h.Len(), 3; got != want {
		t.Errorf("Len = %d; want %d", got, want)
	}

	// Test Less (min-heap property)
	if !h.Less(2, 0) {
		t.Errorf("Less(2,0) = false; want true (fee %f < %f)", h[2].TotalFee, h[0].TotalFee)
	}

	// Test Swap
	h.Swap(0, 2)
	if h[0] != c || h[2] != a {
		t.Errorf("Swap failed; got h[0]=%v, h[2]=%v; want h[0]=%v, h[2]=%v", h[0], h[2], c, a)
	}
}

func TestPushPop(t *testing.T) {
	h := &TxHeap{}
	heap.Init(h)

	fees := []float64{30, 10, 20}
	for _, f := range fees {
		heap.Push(h, &Tx{TotalFee: f})
	}

	// Expected pop order: 10, 20, 30
	expected := []float64{10, 20, 30}
	for _, want := range expected {
		item := heap.Pop(h).(*Tx)
		if got := item.TotalFee; got != want {
			t.Errorf("Pop = %f; want %f", got, want)
		}
	}

	// After popping all, heap should be empty
	if got := h.Len(); got != 0 {
		t.Errorf("Len after pops = %d; want 0", got)
	}
}

func BenchmarkTxHeapPushPop(b *testing.B) {
	h := &TxHeap{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx := &Tx{TotalFee: float64(i)}
		heap.Push(h, tx)
	}
	for h.Len() > 0 {
		heap.Pop(h)
	}
}
