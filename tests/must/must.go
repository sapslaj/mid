package must

type ErrorHandlerFunc func(error)

var ErrorHandler ErrorHandlerFunc = func(err error) { panic(err) }

func Must0(err error) {
	if err != nil {
		ErrorHandler(err)
	}
}

func Must1[A any](a A, err error) A {
	if err != nil {
		ErrorHandler(err)
	}
	return a
}

func Must2[A any, B any](a A, b B, err error) (A, B) {
	if err != nil {
		ErrorHandler(err)
	}
	return a, b
}
