package compilers

import "errors"

var (
	ErrBadRequest error = errors.New("bad request")
	ErrCompiler   error = errors.New("compiler error")
	ErrProgram    error = errors.New("program error")
)
