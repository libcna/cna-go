package framework

// Iterator is the Go language adapter for an XNA IEnumerator<T>. Next returns
// the next value, whether a value was produced, and any enumeration failure.
// A collection mutation after an iterator is created is reported as an error.
type Iterator[T any] interface {
	Next() (value T, ok bool, err error)
}
