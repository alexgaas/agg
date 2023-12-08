package buffer

import (
	"errors"
)

type DataBuffer struct {
	buf [][]string
	off int
}

var ErrTooLarge = errors.New("DataBuffer: too large")

const maxInt = int(^uint(0) >> 1)

func (b *DataBuffer) empty() bool { return len(b.buf) <= b.off }

func (b *DataBuffer) Len() int { return len(b.buf) - b.off }

func (b *DataBuffer) Cap() int { return cap(b.buf) }

func (b *DataBuffer) Truncate(n int) {
	if n == 0 {
		b.Reset()
		return
	}
	if n < 0 || n > b.Len() {
		panic("DataBuffer: truncation out of range")
	}
	b.buf = b.buf[:b.off+n]
}

func (b *DataBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

func (b *DataBuffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); n <= cap(b.buf)-l {
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

func makeSlice(n int) [][]string {
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	return make([][]string, n)
}

func (b *DataBuffer) grow(n int) int {
	m := b.Len()
	if m == 0 && b.off != 0 {
		b.Reset()
	}
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}

	c := cap(b.buf)
	if n <= c/2-m {
		copy(b.buf, b.buf[b.off:])
	} else if c > maxInt-c-n {
		panic(ErrTooLarge)
	} else {
		buf := makeSlice(2*c + n)
		copy(buf, b.buf[b.off:])
		b.buf = buf
	}
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}

func (b *DataBuffer) WriteLine(s []string) error {
	m, ok := b.tryGrowByReslice(1)
	if !ok {
		m = b.grow(1)
	}
	b.buf[m] = s
	return nil
}

func (b *DataBuffer) Next(n int) [][]string {
	m := b.Len()
	if n > m {
		n = m
	}
	data := b.buf[b.off : b.off+n]
	b.off += n
	return data
}
