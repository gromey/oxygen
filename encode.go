package oxygen

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

const marshalError = "encode data from"

// Marshal encodes the value v and returns the encoded data.
// If v is nil, Marshal returns an encoder error.
func (e *engine[T]) Marshal(v any) (out []byte, err error) {
	s := e.newEncodeState()
	defer encodeStatePool.Put(s)

	s.marshal(v)
	return s.Bytes(), s.err
}

type encodeState[T any] struct {
	*engine[T]
	context[T]
	*bytes.Buffer // accumulated output
	scratch       [64]byte
}

var encodeStatePool sync.Pool

func (e *engine[T]) newEncodeState() *encodeState[T] {
	if p := encodeStatePool.Get(); p != nil {
		s := p.(*encodeState[T])
		s.Reset()
		s.err = nil
		return s
	}

	s := &encodeState[T]{engine: e, Buffer: new(bytes.Buffer)}
	s.field = new(field[T])
	return s
}

func (s *encodeState[T]) marshal(v any) {
	if err := s.reflectValue(reflect.ValueOf(v)); err != nil {
		if !errors.Is(err, errExist) {
			s.setError(s.name, marshalError, err)
		}
		s.Reset()
	}
}

func (s *encodeState[T]) reflectValue(v reflect.Value) error {
	return s.cachedCoders(v.Type()).encoderFunc(s, v)
}

type encoderFunc[T any] func(*encodeState[T], reflect.Value) error

func valueFromPtr(v reflect.Value) reflect.Value {
	if v.IsNil() {
		v = reflect.New(v.Type().Elem())
	}
	return v.Elem()
}

func (f *structFields[T]) encode(s *encodeState[T], v reflect.Value, wrap bool) (err error) {
	var sep bool

	s.structName = v.Type().Name()

	if wrap {
		s.Write(s.structOpener)
	}

	for _, s.field = range *f {
		rv := v.Field(s.field.index)

		// Ignore the field if empty values can be omitted.
		if s.field.omitempty && isEmptyValue(rv) {
			continue
		}

		if sep {
			s.Write(s.valueSeparator)
		}
		sep = s.separate

		if s.field.embedded != nil {
			if err = s.field.embedded.encode(s, valueFromPtr(rv), false); err != nil {
				return
			}
			continue
		}

		if err = s.field.functions.encoderFunc(s, rv); err != nil {
			return
		}
	}

	if wrap {
		s.Write(s.structCloser)
	}

	return
}

func marshallerEncoder[T any](s *encodeState[T], v reflect.Value) error {
	tmp := reflect.ValueOf(v.Interface())
	v = reflect.New(v.Type())
	v.Elem().Set(tmp)

	f, ok := s.IsMarshaller(v)
	if !ok {
		return nil
	}

	p, err := f()
	if err != nil {
		return err
	}

	return s.Encode(s.field.name, s.field.tag, p, s.Buffer)
}

func boolEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendBool(s.scratch[:0], v.Bool()), s.Buffer)
}

func intEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendInt(s.scratch[:0], v.Int(), 10), s.Buffer)
}

func uintEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendUint(s.scratch[:0], v.Uint(), 10), s.Buffer)
}

func floatEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendFloat(s.scratch[:0], v.Float(), 'g', -1, bitSize(v.Kind())), s.Buffer)
}

//func arrayEncoder[T any](s *encodeState[T], v reflect.Value) error {
//	return nil
//}

func interfaceEncoder[T any](s *encodeState[T], v reflect.Value) error {
	if v.IsNil() {
		s.err = ErrNilInterface
		return errExist
	}
	return s.reflectValue(v.Elem())
}

//func mapEncoder[T any](s *encodeState[T], v reflect.Value) error {
//	return nil
//}

func pointerEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.reflectValue(valueFromPtr(v))
}

func bytesEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, v.Bytes(), s.Buffer)
}

//func sliceEncoder[T any](s *encodeState[T], v reflect.Value) error {
//	return nil
//}

func stringEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, append(s.scratch[:0], v.String()...), s.Buffer)
}

func structEncoder[T any](s *encodeState[T], v reflect.Value) error {
	f := s.cachedFields(v.Type())
	return f.encode(s, reflect.ValueOf(v.Interface()), s.wrap)
}

func unsupportedTypeEncoder[T any](s *encodeState[T], _ reflect.Value) error {
	s.err = ErrNotSupportType
	return errExist
}

func invalidTagEncoder[T any](tag string, err error) encoderFunc[T] {
	return func(s *encodeState[T], _ reflect.Value) error {
		s.err = fmt.Errorf("%s: tag %s of struct field %s.%s: %w", s.name, tag, s.structName, s.field.name, err)
		return nil
	}
}
