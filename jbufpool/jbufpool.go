package jbufpool

import (
	"io"

	"github.com/valyala/bytebufferpool"
)

type BytebufPoolConsumeRoutine struct {
	dataBufChan chan *bytebufferpool.ByteBuffer
	callback    func(interface{})
}

func NewBytebufPoolConsumeRoutine(chanSize int, callback func(interface{})) io.Writer {
	nw := &BytebufPoolConsumeRoutine{
		dataBufChan: make(chan *bytebufferpool.ByteBuffer, chanSize),
		callback:    callback,
	}
	go nw.consumeRoutine()
	return nw
}

func (r *BytebufPoolConsumeRoutine) Write(data []byte) (int, error) {
	buf := bytebufferpool.Get()

	n, err := buf.Write(data)
	if err != nil {
		return 0, err
	}
	r.dataBufChan <- buf
	return n, nil
}

func (r *BytebufPoolConsumeRoutine) consumeRoutine() {
	for {
		byteBuf := <-(r.dataBufChan)
		r.callback(byteBuf.Bytes())
		bytebufferpool.Put(byteBuf)
	}
}

// type RingBufRoutine struct {
// 	dataBufChan chan []byte
// 	ringbuf     []byte
// 	curOffset   uint64
// 	ringSize    uint64 //MB
// 	callback    func(data []byte)
// }

// func NewRingBufRoutine(chanSize int, ringSize int, callback func(data []byte)) *RingBufRoutine {
// 	nw := &RingBufRoutine{
// 		dataBufChan: make(chan []byte, chanSize),
// 		ringSize:    uint64(ringSize * 1024 * 1024),
// 		callback:    callback,
// 	}
// 	nw.ringbuf = make([]byte, nw.ringSize)
// 	go nw.consumeRouting()
// 	return nw
// }

// func (r *RingBufRoutine) Put(data []byte) {
// 	if uint64(len(data)) >= r.ringSize {
// 		return
// 	}

// 	if r.curOffset+uint64(len(data)) > r.ringSize {
// 		r.curOffset = 0
// 	}

// 	tmp := r.ringbuf[r.curOffset : r.curOffset+uint64(len(data))]
// 	copy(tmp, data)
// 	r.curOffset += uint64(len(data))
// 	if r.curOffset >= r.ringSize {
// 		r.curOffset = 0
// 	}

// 	r.dataBufChan <- tmp
// }

// func (r *RingBufRoutine) consumeRouting() {
// 	for {
// 		data := <-(r.dataBufChan)
// 		r.callback(data)
// 	}
// }
