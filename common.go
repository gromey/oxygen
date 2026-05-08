package oxygen

import (
	"errors"
	"fmt"
	"reflect"
)

type buffer struct {
	b []byte
}

func (b *buffer) Write(p []byte) (int, error) {
	b.b = append(b.b, p...)
	return len(p), nil
}

func (b *buffer) WriteByte(c byte) error {
	b.b = append(b.b, c)
	return nil
}

func (b *buffer) WriteString(s string) (int, error) {
	b.b = append(b.b, s...)
	return len(s), nil
}

func (b *buffer) Bytes() []byte  { return b.b }
func (b *buffer) Len() int       { return len(b.b) }
func (b *buffer) Reset()         { b.b = b.b[:0] }
func (b *buffer) String() string { return string(b.b) }

var (
	errExist = errors.New("exist")

	ErrNotSupportType      = errors.New("cannot support type")
	ErrNilInterface        = errors.New("interface is nil")
	ErrPointerToUnexported = errors.New("cannot set embedded pointer to unexported struct")
	ErrInvalidFormat       = errors.New("the raw data has an invalid format for an object value")
)

func bitSize(v reflect.Kind) int {
	switch v {
	case reflect.Int8, reflect.Uint8:
		return 8
	case reflect.Int16, reflect.Uint16:
		return 16
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 32
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 64
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		return 32 << (^uint(0) >> 63)
	default:
		return 0
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return v.IsZero()
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return !v.IsValid()
	}
}

type context[T any] struct {
	structName string
	field      *field[T]
	err        error
}

func (c *context[T]) setError(tagName, state string, err error) {
	err = unwrapErr(err)
	if c.structName == "" {
		c.err = fmt.Errorf("%s: cannot %s Go value of type %s: %w", tagName, state, c.field.typ, err)
	} else {
		c.err = fmt.Errorf("%s: cannot %s Go struct field %s.%s of type %s: %w", tagName, state, c.structName, c.field.name, c.field.typ, err)
	}
}

func unwrapErr(err error) error {
	if ew := errors.Unwrap(err); ew != nil {
		return ew
	}
	return err
}

func unPoint(t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Pointer {
		return t
	}
	return unPoint(t.Elem())
}
