package oxygen

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

const marshalError = "encode data from"

// Marshal encodes the value v and returns the encoded data.
// If v is nil, Marshal returns an encoder error.
func (e *engine[T]) Marshal(v any) ([]byte, error) {
	s := e.newEncodeState()
	defer e.encStatePool.Put(s)

	if s.marshal(v); s.err != nil {
		return nil, s.err
	}

	buf := s.Bytes()
	out := make([]byte, len(buf))
	copy(out, buf)
	return out, nil
}

type encodeState[T any] struct {
	*engine[T]
	context[T]
	buffer
	fieldBuf field[T]
	scratch  [64]byte
}

func (e *engine[T]) newEncodeState() *encodeState[T] {
	if p := e.encStatePool.Get(); p != nil {
		s := p.(*encodeState[T])
		s.fieldBuf = field[T]{}
		s.field = &s.fieldBuf
		s.err = nil
		s.Reset()
		return s
	}

	s := &encodeState[T]{engine: e}
	s.field = &s.fieldBuf
	return s
}

func (s *encodeState[T]) marshal(v any) {
	if err := s.reflectValue(reflect.ValueOf(v)); err != nil {
		if !errors.Is(err, errExist) {
			if s.field.typ == nil {
				s.field.typ = unPoint(reflect.TypeOf(v))
			}
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
	if v.Kind() != reflect.Pointer {
		return v
	}
	if v.IsNil() {
		v = reflect.New(v.Type().Elem())
	}
	return v.Elem()
}

func (f *structFields[T]) encode(s *encodeState[T], v reflect.Value, wrap bool) (err error) {
	var sep bool

	if wrap {
		if _, err = s.Write(s.structOpener); err != nil {
			return err
		}
	}

	for _, s.field = range *f {
		rv := v.Field(s.field.index)

		// Ignore the field if empty values can be omitted.
		if s.field.omitempty && isEmptyValue(rv) {
			continue
		}

		if sep {
			if _, err = s.Write(s.valueSeparator); err != nil {
				return err
			}
		}
		sep = s.separate

		if s.field.embedded != nil {
			if err = s.field.embedded.encode(s, valueFromPtr(rv), false); err != nil {
				return
			}
			continue
		}

		s.structName = v.Type().Name()
		if err = s.field.functions.encoderFunc(s, rv); err != nil {
			return
		}
	}

	if wrap {
		if _, err = s.Write(s.structCloser); err != nil {
			return err
		}
	}

	return
}

func marshallerEncoder[T any](s *encodeState[T], v reflect.Value) error {
	pv := reflect.New(v.Type())
	pv.Elem().Set(v)

	f, ok := s.IsMarshaller(pv)
	if !ok {
		return nil
	}

	p, err := f()
	if err != nil {
		return err
	}

	return s.Encode(s.field.name, s.field.tag, p, &s.buffer)
}

func boolEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendBool(s.scratch[:0], v.Bool()), &s.buffer)
}

func intEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendInt(s.scratch[:0], v.Int(), 10), &s.buffer)
}

func uintEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendUint(s.scratch[:0], v.Uint(), 10), &s.buffer)
}

func floatEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, strconv.AppendFloat(s.scratch[:0], v.Float(), 'g', -1, bitSize(v.Kind())), &s.buffer)
}

//func arrayEncoder[T any](s *encodeState[T], v reflect.Value) error {
//	return nil
//}

func interfaceEncoder[T any](s *encodeState[T], v reflect.Value) error {
	if v.IsNil() {
		return nil
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
	return s.Encode(s.field.name, s.field.tag, v.Bytes(), &s.buffer)
}

//func sliceEncoder[T any](s *encodeState[T], v reflect.Value) error {
//	return nil
//}

func stringEncoder[T any](s *encodeState[T], v reflect.Value) error {
	return s.Encode(s.field.name, s.field.tag, append(s.scratch[:0], v.String()...), &s.buffer)
}

func structEncoder[T any](s *encodeState[T], v reflect.Value) error {
	f := s.cachedFields(v.Type())
	return f.encode(s, v, s.wrap)
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
