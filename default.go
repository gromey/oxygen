package oxygen

type Default[T any] struct{}

func (*Default[T]) Parse(string, *T) (bool, error) {
	return false, nil
}

func (*Default[T]) Encode(_ string, _ *T, in []byte, out Writer) (err error) {
	_, err = out.Write(in)
	return
}

func (*Default[T]) f() {}
