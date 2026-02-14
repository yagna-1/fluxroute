package channel

// NewBufferedResultChannel allocates a buffered channel with at least size 1.
func NewBufferedResultChannel[T any](size int) chan T {
	if size <= 0 {
		size = 1
	}
	return make(chan T, size)
}
