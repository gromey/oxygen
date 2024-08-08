package oxygen

import (
	"reflect"
	"sync"
)

// Engine represents the main functions that the package implements.
type Engine interface {
	// Marshal encodes the value v and returns the encoded data.
	Marshal(v any) ([]byte, error)
	// Unmarshal decodes the encoded data and stores the result in the value pointed to by v.
	Unmarshal(data []byte, v any) error
}

type Writer interface {
	Write(p []byte) (n int, err error)
	WriteByte(c byte) error
	WriteString(s string) (n int, err error)
}

// Tag describes what functions an entity should implement to use when creating a new Engine entity.
// The entity must include the oxygen.Default that implements following default Parse method,
// so it may not implement this method.
type Tag[T any] interface {
	// Parse gets a tagValue string, parses the tagValue into tag *T,
	// returns a flag indicating that the field is skipped if it's empty,
	// and if parsing fails, it returns an error.
	Parse(tagValue string, tag *T) (bool, error)
	// Encode takes encoded data and performs secondary encoding.
	// It's a mandatory function.
	Encode(fieldName string, tag *T, in []byte, out Writer) error
	// Decode takes the raw encoded data and performs a primary decode.
	// It's a mandatory function.
	Decode(fieldName string, tag *T, in []byte, out Writer) error
	// IsMarshaller attempts to cast the value to a Marshaller interface,
	// if so, returns a marshal function.
	IsMarshaller(v reflect.Value) (func() ([]byte, error), bool)
	// IsUnmarshaler attempts to cast the value to an Unmarshaler interface,
	// if so, returns an unmarshal function.
	IsUnmarshaler(v reflect.Value) (func([]byte) error, bool)

	f()
}

type Config struct {
	// Name of the tag.
	Name string
	// StructOpener a byte array that denotes the beginning of a structure.
	// Will be automatically added when encoding.
	StructOpener []byte
	// StructCloser a byte array that denotes the end of a structure.
	// Will be automatically added when encoding.
	StructCloser []byte
	// UnwrapWhenDecoding this flag tells the library whether to remove the StructOpener and StructCloser bytes of a structure.
	UnwrapWhenDecoding bool
	// ValueSeparator a byte array separating values.
	// Will be automatically added when encoding.
	ValueSeparator []byte
	// RemoveSeparatorWhenDecoding this flag tells the library whether to remove the ValueSeparator.
	RemoveSeparatorWhenDecoding bool
	// Marshaller is used to check if a type implements a type of the Marshaller interface.
	Marshaller reflect.Type
	// Unmarshaler is used to check if a type implements a type of the Unmarshaler interface.
	Unmarshaler reflect.Type
}

// New returns a new entity that implements the Engine interface.
func New[T any](tag Tag[T], cfg Config) Engine {
	return &engine[T]{
		Tag:             tag,
		name:            cfg.Name,
		wrap:            len(cfg.StructOpener) != 0 || len(cfg.StructCloser) != 0,
		removeWrapper:   (len(cfg.StructOpener) != 0 || len(cfg.StructCloser) != 0) && cfg.UnwrapWhenDecoding,
		separate:        len(cfg.ValueSeparator) != 0,
		removeSeparator: len(cfg.ValueSeparator) != 0 && cfg.RemoveSeparatorWhenDecoding,
		structOpener:    cfg.StructOpener,
		structCloser:    cfg.StructCloser,
		valueSeparator:  cfg.ValueSeparator,
		marshaller:      cfg.Marshaller,
		unmarshaler:     cfg.Unmarshaler,
	}
}

type engine[T any] struct {
	Tag[T]
	name                                           string
	wrap, removeWrapper, separate, removeSeparator bool
	structOpener, structCloser, valueSeparator     []byte
	marshaller, unmarshaler                        reflect.Type
}

type coders[T any] struct {
	encoderFunc[T]
	decoderFunc[T]
}

var coderCache sync.Map // map[reflect.Type]*coders[T]

// cachedCoders is like typeCoders but uses a cache to avoid repeated work.
func (e *engine[T]) cachedCoders(t reflect.Type) *coders[T] {
	if c, ok := coderCache.Load(t); ok {
		return c.(*coders[T])
	}

	c, _ := coderCache.LoadOrStore(t, e.typeCoders(t))
	return c.(*coders[T])
}

// typeCoders returns coders for a type.
func (e *engine[T]) typeCoders(t reflect.Type) *coders[T] {
	f := new(coders[T])
	switch t.Kind() {
	case reflect.Bool:
		f.encoderFunc = boolEncoder[T]
		f.decoderFunc = boolDecoder[T]
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.encoderFunc = intEncoder[T]
		f.decoderFunc = intDecoder[T]
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		f.encoderFunc = uintEncoder[T]
		f.decoderFunc = uintDecoder[T]
	case reflect.Float32, reflect.Float64:
		f.encoderFunc = floatEncoder[T]
		f.decoderFunc = floatDecoder[T]
	//case reflect.Array:
	//	f.encoderFunc = arrayEncoder[T]
	//	f.decoderFunc = arrayDecoder[T]
	case reflect.Interface:
		f.encoderFunc = interfaceEncoder[T]
		f.decoderFunc = interfaceDecoder[T]
	//case reflect.Map:
	//	f.encoderFunc = mapEncoder[T]
	//	f.decoderFunc = mapDecoder[T])
	case reflect.Pointer:
		f.encoderFunc = pointerEncoder[T]
		f.decoderFunc = pointerDecoder[T]
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			f.encoderFunc = bytesEncoder[T]
			f.decoderFunc = bytesDecoder[T]
		} else {
			f.encoderFunc = unsupportedTypeEncoder[T]
			f.decoderFunc = unsupportedTypeDecoder[T]
		}
	case reflect.String:
		f.encoderFunc = stringEncoder[T]
		f.decoderFunc = stringDecoder[T]
	case reflect.Struct:
		f.encoderFunc = structEncoder[T]
		f.decoderFunc = structDecoder[T]
	default:
		f.encoderFunc = unsupportedTypeEncoder[T]
		f.decoderFunc = unsupportedTypeDecoder[T]
	}

	if t.Kind() != reflect.Pointer {
		p := reflect.PointerTo(t)
		if p.Implements(e.marshaller) {
			f.encoderFunc = marshallerEncoder[T]
		}
		if p.Implements(e.unmarshaler) {
			f.decoderFunc = unmarshalerDecoder[T]
		}
	}

	return f
}

// field represents a single field found in a struct.
type field[T any] struct {
	index     int
	name      string
	typ       reflect.Type
	tag       *T
	omitempty bool
	functions *coders[T]
	embedded  structFields[T]
}

type structFields[T any] []*field[T]

var fieldCache sync.Map // map[reflect.Type]structFields[T]

// cachedFields is like typeFields but uses a cache to avoid repeated work.
func (e *engine[T]) cachedFields(t reflect.Type) structFields[T] {
	if c, ok := fieldCache.Load(t); ok {
		return c.(structFields[T])
	}
	c, _ := fieldCache.LoadOrStore(t, e.typeFields(t))
	return c.(structFields[T])
}

// typeFields returns a list of fields that the encoder/decoder should recognize for the given type.
func (e *engine[T]) typeFields(t reflect.Type) structFields[T] {
	var err error

	fs := make(structFields[T], 0, t.NumField())

	// Scan type for fields to encode/decode.
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		ft := sf.Type

		f := &field[T]{
			index: i,
			name:  sf.Name,
			typ:   ft,
		}

		if sf.Anonymous {
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			// Ignore embedded fields of unexported non-struct types.
			if !sf.IsExported() && ft.Kind() != reflect.Struct {
				continue
			}

			// Do not ignore embedded fields of unexported struct types since they may have exported fields.
			f.embedded = e.cachedFields(ft)

			if f.embedded == nil {
				continue
			}

			fs = append(fs, f)
			continue
		} else if !sf.IsExported() {
			// Ignore unexported non-embedded fields.
			continue
		}

		if tag, ok := sf.Tag.Lookup(e.name); ok {
			// Ignore the field if the tag has a skip value.
			if tag == "-" {
				continue
			}

			f.tag = new(T)
			if f.omitempty, err = e.Parse(tag, f.tag); err != nil {
				f.functions = &coders[T]{
					encoderFunc: invalidTagEncoder[T](tag, err),
					decoderFunc: invalidTagDecoder[T](tag, err),
				}
				return append(fs, f)
			}
		}

		f.functions = e.cachedCoders(ft)
		fs = append(fs, f)
	}

	return fs
}
