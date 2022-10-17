package jroutinepool

import (
	"io"
	"sync/atomic"
)

type MultipleRoutineConsumePool struct {
	rbr          []io.Writer
	currfcOffset uint32
	poolNum      uint32
}

func NewMultipleRoutineConsumePool(poolNum int, w func() io.Writer) *MultipleRoutineConsumePool {

	nw := &MultipleRoutineConsumePool{}
	for i := 0; i < poolNum; i++ {
		nw.rbr = append(nw.rbr, w())
	}
	nw.poolNum = uint32(poolNum)
	return nw
}

func (rp *MultipleRoutineConsumePool) Write(data []byte) (int, error) {

	if rp.currfcOffset >= rp.poolNum {
		atomic.CompareAndSwapUint32(&rp.currfcOffset, rp.currfcOffset, 0)
	}

	n, err := rp.rbr[rp.currfcOffset].Write(data)
	atomic.AddUint32(&rp.currfcOffset, 1)

	return n, err
}
