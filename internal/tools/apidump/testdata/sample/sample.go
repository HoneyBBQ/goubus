// Package sample exercises every construct apidump renders.
package sample

import (
	"context"
	"fmt"
	"time"

	"github.com/honeybbq/goubus/v2/internal/tools/apidump/testdata/sample/internal/hidden"
)

// Constants: typed, untyped, float and string.
const (
	Answer         = 42
	Pi             = 3.14159
	Greeting       = "hello \"world\""
	Short    Level = 1
	Long     Level = 2
)

// Level is a defined integer type.
type Level int

// Level methods: one on the value, one on the pointer.
func (l Level) String() string { return fmt.Sprint(int(l)) }

// Set changes the level.
func (l *Level) Set(v int) { *l = Level(v) }

// Variables.
var (
	ErrFailed error
	Timeout = time.Second
)

// Map is a generic function.
func Map[T, U any](in []T, f func(T) U) []U {
	_, _ = in, f

	return nil
}

// Dial has a variadic parameter and two results.
func Dial(ctx context.Context, addr string, opts ...Option) (*Conn, error) {
	_, _, _ = ctx, addr, opts

	return nil, nil
}

// Option is a closed interface: nothing outside this package can implement it.
type Option interface {
	apply(*Conn)
	// Name names the option.
	Name() string
}

// Reader is an open interface that embeds another.
type Reader interface {
	fmt.Stringer
	// Read reads.
	Read(p []byte) (int, error)
}

// Number is a constraint.
type Number interface {
	~int | ~float64
	// Double doubles.
	Double() int
}

// base is embedded unexported: its exported members are promoted.
type base struct {
	// ID is promoted into Conn.
	ID int
}

// Hello is promoted into Conn.
func (b *base) Hello() string { return "" }

// Conn is a struct with tags, an exported embedded type and an unexported one.
type Conn struct {
	base
	Meta
	// Addr is the address.
	Addr string `json:"addr"`
	// Level is the level.
	Level  Level
}

// Meta is embedded exported: its methods are promoted into Conn.
type Meta struct{}

// Created says when.
func (m Meta) Created() time.Time { return time.Time{} }

// Close closes.
func (c *Conn) Close() error { return nil }

// Stack is a generic type.
type Stack[T any] struct {
	// Items are the items.
	Items []T
}

// Push pushes.
func (s *Stack[T]) Push(v T) { _ = v }

// Len counts.
func (s Stack[T]) Len() int { return len(s.Items) }

// Handler is a function type.
type Handler func(ctx context.Context, name string) error

// Names is a map type.
type Names map[string][]string

// Duration is an alias of a standard type.
type Duration = time.Duration

// Secret is an alias of an internal type: its members are reachable through it.
type Secret = hidden.Secret

// Leaky exposes types a caller cannot name.
func Leaky(s *hidden.Secret) *conn {
	_ = s

	return nil
}

// conn is unexported.
type conn struct{}

// Field exposes an unexported type in a struct field.
type Field struct {
	// Inner cannot be named.
	Inner conn
	// Slice hides one in a slice.
	Slice []conn
}
