package channel

type Channel interface {
	InitChannel() error
	Send(msg string) error
	Close() error
}
