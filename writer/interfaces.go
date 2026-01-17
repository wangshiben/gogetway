package writer

type Flushable interface {
	Flush() error
}
