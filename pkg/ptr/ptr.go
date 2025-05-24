package ptr

import "reflect"

func Of[T any](v T) *T {
	return &v
}

func FromDefault[T any](v *T, def T) T {
	if v != nil {
		return *v
	}
	return def
}

func From[T any](v *T) T {
	var def T
	return FromDefault(v, def)
}

func ToAny[T any](a *T) *any {
	if a == nil {
		return nil
	}
	b := any(*a)
	return &b
}

func SlicesOf[T any](v []T) []*T {
	s := make([]*T, len(v))
	for i := range v {
		s[i] = &v[i]
	}
	return s
}

func SlicesFrom[T any](v []*T) []T {
	s := make([]T, len(v))
	for i := range v {
		s[i] = *v[i]
	}
	return s
}

func AnyNonNils(vs ...any) bool {
	for _, v := range vs {
		if v != nil && !reflect.ValueOf(v).IsNil() {
			return true
		}
	}
	return false
}
