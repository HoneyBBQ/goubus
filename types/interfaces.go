package types

type Transport interface {
	Call(service, method string, data any) (Result, error)
	Close() error
}

type Result interface {
	Unmarshal(target any) error
}
