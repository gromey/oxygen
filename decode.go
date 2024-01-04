package oxygen

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"unicode"
)

// Proper usage of a sync.Pool requires each entry to have approximately
// the same memory cost. To obtain this property when the stored type
// contains a variably-sized buffer, we add a hard limit on the maximum buffer
// to place back in the pool.
//
// See https://golang.org/issue/23199
const maxSize = 1 << 16 // 64KiB

const unmarshalError = "decode data into"

// Unmarshal decodes the encoded data and stores the result in the value pointed to by v.
// If v is nil or not a pointer, Unmarshal returns a decoder error.
func (e *engine[T]) Unmarshal(data []byte, v any) (err error) {
	if t := reflect.ValueOf(v).Kind(); t != reflect.Pointer {
		return fmt.Errorf("%s: Unmarshal(non-pointer %s)", e.name, t)
	}

	s := e.newDecodeState()
	defer putDecodeState(s)

	s.data = append(s.data, data...)

	s.unmarshal(v)
	return s.err
}

type decodeState[T any] struct {
	*engine[T]
	context[T]
	*bytes.Buffer
	data []byte // copy of input
}

var decodeStatePool sync.Pool

func (e *engine[T]) newDecodeState() *decodeState[T] {
	if p := decodeStatePool.Get(); p != nil {
		s := p.(*decodeState[T])
		s.err = nil
		s.Reset()
		s.data = s.data[:0]
		return s
	}

	s := &decodeState[T]{engine: e, Buffer: new(bytes.Buffer), data: make([]byte, 0, 512)}
	s.field = new(field[T])
	return s
}

func putDecodeState[T any](s *decodeState[T]) {
	if cap(s.data) <= maxSize {
		decodeStatePool.Put(s)
	}
}

func (s *decodeState[T]) unmarshal(v any) {
	if err := s.reflectValue(reflect.ValueOf(v)); err != nil {
		if !errors.Is(err, errExist) {
			if s.field.typ == nil {
				s.field.typ = unPoint(reflect.TypeOf(v))
			}
			s.setError(s.name, unmarshalError, err)
		}
	}
}

func (s *decodeState[T]) reflectValue(v reflect.Value) error {
	return s.cachedCoders(v.Type()).decoderFunc(s, v)
}

type decoderFunc[T any] func(*decodeState[T], reflect.Value) error

func (s *decodeState[T]) removePrefixBytes(b []byte) error {
	if !bytes.HasPrefix(s.data, b) {
		s.err = fmt.Errorf("%s: %w", s.name, ErrInvalidFormat)
		return errExist
	}
	s.data = s.data[len(b):]
	return nil
}

func (f *structFields[T]) decode(s *decodeState[T], v reflect.Value, unwrap bool) (err error) {
	var sep bool

	if unwrap {
		if err = s.removePrefixBytes(s.structOpener); err != nil {
			return
		}
	}

	for _, s.field = range *f {
		if s.data = bytes.TrimRightFunc(s.data, unicode.IsSpace); s.data == nil || unwrap && bytes.HasPrefix(s.data, s.structCloser) {
			break
		}

		if sep {
			if err = s.removePrefixBytes(s.valueSeparator); err != nil {
				return
			}
		}
		sep = s.removeSeparator

		s.Reset()
		rv := v.Field(s.field.index)

		if s.field.embedded != nil {
			if rv.Kind() == reflect.Pointer {
				if rv.IsNil() {
					s.err = fmt.Errorf("%s: %w: %s", s.name, ErrPointerToUnexported, rv.Type().Elem())
					return errExist
				}
				rv = rv.Elem()
			}

			if err = s.field.embedded.decode(s, rv, false); err != nil {
				return
			}
			continue
		}

		s.structName = v.Type().Name()
		if err = s.field.functions.decoderFunc(s, rv); err != nil {
			return
		}
	}

	if unwrap {
		if err = s.removePrefixBytes(s.structCloser); err != nil {
			return
		}
	}

	return
}

func unmarshalerDecoder[T any](s *decodeState[T], v reflect.Value) error {
	rv := reflect.New(v.Type())

	f, ok := s.IsUnmarshaler(rv)
	if !ok {
		return nil
	}

	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}

	if err := f(s.Bytes()); err != nil {
		return err
	}

	v.Set(rv.Elem())
	return nil
}

func boolDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	r, err := strconv.ParseBool(s.String())
	v.SetBool(r)
	return err
}

func intDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	r, err := strconv.ParseInt(s.String(), 10, bitSize(v.Kind()))
	v.SetInt(r)
	return err
}

func uintDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	r, err := strconv.ParseUint(s.String(), 10, bitSize(v.Kind()))
	v.SetUint(r)
	return err
}

func floatDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	r, err := strconv.ParseFloat(s.String(), bitSize(v.Kind()))
	v.SetFloat(r)
	return err
}

//func arrayDecoder[T any](s *decodeState[T], v reflect.Value) error {
//	return nil
//}

func interfaceDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if v.IsNil() {
		s.err = ErrNilInterface
		return errExist
	}
	return s.reflectValue(v.Elem())
}

//func mapDecoder[T any](s *decodeState[T], v reflect.Value) error {
//	return nil
//}

func pointerDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if v.IsNil() {
		rv := reflect.New(v.Type().Elem())
		if err := s.reflectValue(rv.Elem()); err != nil {
			return err
		}
		if !isEmptyValue(rv.Elem()) {
			v.Set(rv)
		}
		return nil
	}
	return s.reflectValue(v.Elem())
}

func bytesDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	v.SetBytes(s.Bytes())
	return nil
}

//func sliceDecoder[T any](s *decodeState[T], v reflect.Value) error {
//	return nil
//}

func stringDecoder[T any](s *decodeState[T], v reflect.Value) error {
	if err := s.Decode(s.field.name, s.field.tag, s.data, s); err != nil {
		return err
	}
	if s.Len() == 0 {
		return nil
	}
	v.SetString(s.String())
	return nil
}

func structDecoder[T any](s *decodeState[T], v reflect.Value) error {
	f := s.cachedFields(v.Type())
	return f.decode(s, v, s.wrap)
}

func unsupportedTypeDecoder[T any](s *decodeState[T], _ reflect.Value) error {
	s.err = ErrNotSupportType
	return errExist
}

func invalidTagDecoder[T any](tag string, err error) decoderFunc[T] {
	return func(s *decodeState[T], _ reflect.Value) error {
		s.err = fmt.Errorf("%s: tag %s of struct field %s.%s: %w", s.name, tag, s.structName, s.field.name, err)
		return errExist
	}
}
