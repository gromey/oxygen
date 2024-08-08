package test_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gromey/oxygen/test"
)

func equal(t *testing.T, exp, got interface{}) {
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("Not equal:\nexp: %v\ngot: %v", exp, got)
	}
}

var (
	Bool    bool    = true
	Int     int     = 11
	Int8    int8    = 12
	Int16   int16   = 13
	Int32   int32   = 14
	Int64   int64   = 15
	Uint    uint    = 21
	Uint8   uint8   = 22
	Uint16  uint16  = 23
	Uint32  uint32  = 24
	Uint64  uint64  = 25
	Uintptr uintptr = 26
	Float32 float32 = 33.3
	Float64 float64 = 4.44
	Str     string  = "test"
	Sl              = []byte("TEST")
)

type baseTypes struct {
	Bool     bool     `test:"5, ,l"`
	Int      int      `test:"4,0,r"`
	Int8     int8     `test:"4,0,r"`
	Int16    int16    `test:"4,0,r"`
	Int32    int32    `test:"4,0,r"`
	Int64    int64    `test:"4,0,r"`
	Uint     uint     `test:"4,0,r"`
	Uint8    uint8    `test:"4,0,r"`
	Uint16   uint16   `test:"4,0,r"`
	Uint32   uint32   `test:"4,0,r"`
	Uint64   uint64   `test:"4,0,r"`
	Uintptr  uintptr  `test:"4,0,r"`
	Float32  float32  `test:"5,0,r"`
	Float64  float64  `test:"5,0,r"`
	Str      string   `test:"10,_,l"`
	Sl       []byte   `test:"4, ,l"`
	PBool    *bool    `test:"5, ,l"`
	PInt     *int     `test:"4,0,r"`
	PInt8    *int8    `test:"4,0,r"`
	PInt16   *int16   `test:"4,0,r"`
	PInt32   *int32   `test:"4,0,r"`
	PInt64   *int64   `test:"4,0,r"`
	PUint    *uint    `test:"4,0,r"`
	PUint8   *uint8   `test:"4,0,r"`
	PUint16  *uint16  `test:"4,0,r"`
	PUint32  *uint32  `test:"4,0,r"`
	PUint64  *uint64  `test:"4,0,r"`
	PUintptr *uintptr `test:"4,0,r"`
	PFloat32 *float32 `test:"5,0,r"`
	PFloat64 *float64 `test:"5,0,r"`
	PStr     *string  `test:"10,_,l"`
	PSl      *[]byte  `test:"4, ,l"`
}

var sbt = baseTypes{
	Bool:     false,
	Int:      99,
	Int8:     98,
	Int16:    97,
	Int32:    96,
	Int64:    95,
	Uint:     89,
	Uint8:    88,
	Uint16:   87,
	Uint32:   86,
	Uint64:   85,
	Uintptr:  84,
	Float32:  77.7,
	Float64:  6.66,
	Str:      "Hel Wor",
	Sl:       []byte("TEST"),
	PBool:    &Bool,
	PInt:     &Int,
	PInt8:    &Int8,
	PInt16:   &Int16,
	PInt32:   &Int32,
	PInt64:   &Int64,
	PUint:    &Uint,
	PUint8:   &Uint8,
	PUint16:  &Uint16,
	PUint32:  &Uint32,
	PUint64:  &Uint64,
	PUintptr: &Uintptr,
	PFloat32: &Float32,
	PFloat64: &Float64,
	PStr:     &Str,
	PSl:      &Sl,
}

type cstStr *string

type wrappedType struct {
	Str cstStr `test:"10,_,l"`
}

var wt = wrappedType{
	Str: cstStr(&Str),
}

var wtEmpty = wrappedType{}

type Inter interface {
	IStr() string
}

type sub struct {
	Str  string  `test:"10,?,l"`
	PStr *string `test:"10,-,r"`
	Ha   string  `test:"-"`
}

func (s sub) IStr() string {
	return s.Str + *s.PStr
}

type interfaceType struct {
	In Inter `bt:"0, ,l"`
}

var it = interfaceType{
	In: &sub{Str: "iii"},
}

var itEmpty = interfaceType{}

type structFields struct {
	Sub  sub
	PSub *sub
}

var sf = structFields{
	Sub: sub{
		Str:  "Sub test",
		PStr: &Str,
	},
	PSub: &sub{
		Str:  "Sub test",
		PStr: &Str,
	},
}

type nestedType struct {
	sub
	I int `test:"4,0,r"`
}

var nt = nestedType{
	sub: sub{
		Str:  "Sub test",
		PStr: &Str,
	},
	I: 7,
}

type nestedPtrType struct {
	*sub
	I int `test:"4,0,r"`
}

var npt = nestedPtrType{
	sub: &sub{
		Str:  "Sub test",
		PStr: &Str,
	},
	I: 7,
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect []byte
		err    error
	}{
		{
			name:   "struct with base types",
			input:  sbt,
			expect: []byte("{false,0099,0098,0097,0096,0095,0089,0088,0087,0086,0085,0084,077.7,06.66,Hel Wor___,TEST,true ,0011,0012,0013,0014,0015,0021,0022,0023,0024,0025,0026,033.3,04.44,test______,TEST}"),
		},
		{
			name:   "pointer to struct with base types",
			input:  &sbt,
			expect: []byte("{false,0099,0098,0097,0096,0095,0089,0088,0087,0086,0085,0084,077.7,06.66,Hel Wor___,TEST,true ,0011,0012,0013,0014,0015,0021,0022,0023,0024,0025,0026,033.3,04.44,test______,TEST}"),
		},
		{
			name:   "struct with a wrapped type",
			input:  wt,
			expect: []byte("{test______}"),
		},
		{
			name:   "empty struct with a wrapped type",
			input:  wtEmpty,
			expect: []byte("{__________}"),
		},
		{
			name:   "struct with an interface",
			input:  it,
			expect: []byte("{{iii???????,----------}}"),
		},
		{
			name:   "struct with nil interface",
			input:  itEmpty,
			expect: []byte("{}"),
		},
		{
			name:   "struct with struct fields",
			input:  sf,
			expect: []byte("{{Sub test??,------test},{Sub test??,------test}}"),
		},
		{
			name:   "struct with nested struct fields",
			input:  nt,
			expect: []byte("{Sub test??,------test,0007}"),
		},
		{
			name:   "struct with nested pointer to struct fields",
			input:  npt,
			expect: []byte("{Sub test??,------test,0007}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := test.Marshal(tt.input)
			if tt.err != nil {
				equal(t, tt.err.Error(), err.Error())
				return
			}
			equal(t, nil, err)
			equal(t, tt.expect, data)
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		output any
		expect any
		err    error
	}{
		{
			name:   "part of fields struct with base types",
			input:  []byte("{true ,0099,0098,0097,0096}"),
			output: new(baseTypes),
			expect: &baseTypes{
				Bool:  true,
				Int:   99,
				Int8:  98,
				Int16: 97,
				Int32: 96,
			},
		},
		{
			name:   "struct with base types",
			input:  []byte("{false,0099,0098,0097,0096,0095,0089,0088,0087,0086,0085,0084,077.7,06.66,Hel Wor___,TEST,true ,0011,0012,0013,0014,0015,0021,0022,0023,0024,0025,0026,033.3,04.44,test______,TEST}"),
			output: new(baseTypes),
			expect: &sbt,
		},
		{
			name:   "struct with a wrapped type",
			input:  []byte("{test______}"),
			output: new(wrappedType),
			expect: &wt,
		},
		{
			name:   "empty struct with a wrapped type",
			input:  []byte("{__________}"),
			output: new(wrappedType),
			expect: &wtEmpty,
		},
		{
			name:   "struct with an interface",
			input:  []byte("{{iii???????,----------}}"),
			output: &interfaceType{In: &sub{}},
			expect: &it,
		},
		{
			name:   "struct with nil interface",
			input:  []byte("{}"),
			output: new(interfaceType),
			expect: &itEmpty,
		},
		{
			name:   "struct with struct fields",
			input:  []byte("{{Sub test??,------test},{Sub test??,------test}}"),
			output: new(structFields),
			expect: &sf,
		},
		{
			name:   "struct with nested struct fields",
			input:  []byte("{Sub test??,------test,0007}"),
			output: new(nestedType),
			expect: &nt,
		},
		{
			name:   "struct with nested pointer to struct fields",
			input:  []byte("{Sub test??,------test,0007}"),
			output: &nestedPtrType{sub: &sub{}},
			expect: &npt,
		},
		{
			name:   "Unmarshal(non-pointer struct)",
			input:  []byte("{Sub test??,------test,0007}"),
			output: nestedPtrType{},
			err:    errors.New("test: Unmarshal(non-pointer struct)"),
		},
		{
			name:   "cannot set embedded pointer to unexported struct",
			input:  []byte("{Sub test??,------test,0007}"),
			output: &nestedPtrType{},
			err:    errors.New("test: cannot set embedded pointer to unexported struct: test_test.sub"),
		},
		{
			name:   "type int: invalid syntax",
			input:  []byte("{Sub test??,------test,00d7}"),
			output: &nestedType{},
			err:    errors.New("test: cannot decode data into Go struct field nestedType.I of type int: invalid syntax"),
		},
		{
			name:   "invalid format for an object value",
			input:  []byte("{Sub test??,------test_,0007}"),
			output: &nestedType{},
			err:    errors.New("test: the raw data has an invalid format for an object value"),
		},
		{
			name:   "invalid format for an object value 2",
			input:  []byte("Sub test??,------test,0007}"),
			output: &nestedType{},
			err:    errors.New("test: the raw data has an invalid format for an object value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := test.Unmarshal(tt.input, tt.output)
			if tt.err != nil {
				equal(t, tt.err.Error(), err.Error())
				return
			}
			equal(t, nil, err)
			equal(t, tt.expect, tt.output)
		})
	}
}

func BenchmarkPrimeNumbers(b *testing.B) {
	input := []byte("{{Sub test??,------test},{Sub test??,------test}}")
	output := new(structFields)
	for i := 0; i < b.N; i++ {
		if err := test.Unmarshal(input, output); err != nil {
			panic(err)
		}
	}
}
