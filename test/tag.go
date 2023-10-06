package test

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gromey/oxygen"
)

var (
	cfg = oxygen.Config{
		StructOpener:                []byte("{"),
		StructCloser:                []byte("}"),
		UnwrapWhenDecoding:          true,
		ValueSeparator:              []byte(","),
		RemoveSeparatorWhenDecoding: true,
		// WARNING: DO NOT DELETE CONFIGURATIONS BELOW!
		Name:        "test",
		Marshaller:  reflect.TypeOf((*Marshaller)(nil)).Elem(),
		Unmarshaler: reflect.TypeOf((*Unmarshaler)(nil)).Elem(),
	}
	test = oxygen.New[tag](&engine{}, cfg)
)

// Marshal encodes the value v and returns the encoded data.
func Marshal(v any) ([]byte, error) {
	return test.Marshal(v)
}

// Unmarshal decodes the encoded data and stores the result in the value pointed to by v.
func Unmarshal(b []byte, v any) error {
	return test.Unmarshal(b, v)
}

type engine struct {
	oxygen.Default[tag]
}

type tag struct {
	Len    int
	Filler byte
	Align  byte
}

// Parse gets a tagValue string, parses the tagValue into tag *tag,
// returns a flag indicating that the field is skipped if it's empty.
func (e *engine) Parse(tagValue string, tag *tag) (omit bool, err error) {
	tagParts := strings.Split(tagValue, ",")

	for i, v := range tagParts {
		switch i {
		case 0:
			if tag.Len, err = strconv.Atoi(v); err != nil {
				return
			}
		case 1:
			if len(v) != 1 {
				return
			}
			tag.Filler = v[0]
		case 2:
			if len(v) != 1 {
				return
			}
			tag.Align = v[0]
		}
	}

	return
}

// Encode takes encoded data and performs secondary encoding to TEST format.
func (e *engine) Encode(_ string, tag *tag, in []byte, out oxygen.Writer) (err error) {
	if tag == nil || len(in) == tag.Len || tag.Len == 0 {
		_, err = out.Write(in)
		return
	}

	if len(in) > tag.Len {
		return fmt.Errorf("data for encoding [%d] more than field length [%d]", len(in), tag.Len)
	}

	if tag.Align == 'l' {
		if _, err = out.Write(in); err != nil {
			return
		}
		for i := 0; i < tag.Len-len(in); i++ {
			if err = out.WriteByte(tag.Filler); err != nil {
				return
			}
		}
	} else {
		for i := 0; i < tag.Len-len(in); i++ {
			if err = out.WriteByte(tag.Filler); err != nil {
				return
			}
		}
		_, err = out.Write(in)
	}

	return
}

// Decode takes the raw encoded data and performs a primary decode from TEST format.
func (e *engine) Decode(_ string, tag *tag, in []byte, out oxygen.Writer) (err error) {
	if tag == nil || tag.Len == 0 {
		_, err = out.Write(in)
		return
	}

	if tag.Align == 'l' {
		_, err = out.Write(bytes.TrimRight(in[:tag.Len], string(tag.Filler)))
	} else {
		_, err = out.Write(bytes.TrimLeft(in[:tag.Len], string(tag.Filler)))
	}
	if err != nil {
		return
	}

	copy(in, in[tag.Len:])

	return
}
