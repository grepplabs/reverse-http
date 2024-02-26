package gost

import (
	"io"
	"sync"
)

const (
	bufferSize = 64 * 1024
)

func NetTransport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	if err := <-errc; err != nil && err != io.EOF {
		return err
	}

	return nil
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)
	_, err := io.CopyBuffer(dst, src, *buf)
	return err
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, bufferSize)
		return &b
	},
}
