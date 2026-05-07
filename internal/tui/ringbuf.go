package tui

// RingBuf is a fixed-capacity FIFO. Push overwrites the oldest
// entry once the buffer is full. Snapshot returns a copy of the
// entries in chronological (oldest-first) order.
type RingBuf[T any] struct {
	buf  []T
	head int
	size int
}

// NewRingBuf returns a buffer of the given capacity.
func NewRingBuf[T any](cap int) *RingBuf[T] {
	return &RingBuf[T]{buf: make([]T, cap)}
}

func (r *RingBuf[T]) Push(v T) {
	if cap(r.buf) == 0 {
		return
	}
	idx := (r.head + r.size) % cap(r.buf)
	if r.size == cap(r.buf) {
		r.head = (r.head + 1) % cap(r.buf)
		r.buf[idx] = v
	} else {
		r.buf[idx] = v
		r.size++
	}
}

func (r *RingBuf[T]) Len() int { return r.size }

func (r *RingBuf[T]) Snapshot() []T {
	out := make([]T, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.head+i)%cap(r.buf)]
	}
	return out
}

// Latest returns up to n most-recent entries, newest first. If n is
// zero or larger than the buffer, the whole buffer is returned.
func (r *RingBuf[T]) Latest(n int) []T {
	if n <= 0 || n > r.size {
		n = r.size
	}
	out := make([]T, n)
	for i := 0; i < n; i++ {
		// newest sits at head+size-1
		idx := (r.head + r.size - 1 - i) % cap(r.buf)
		if idx < 0 {
			idx += cap(r.buf)
		}
		out[i] = r.buf[idx]
	}
	return out
}

func (r *RingBuf[T]) Clear() {
	r.head = 0
	r.size = 0
}
