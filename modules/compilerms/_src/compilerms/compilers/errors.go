package compilers

import "errors"

var (
	ErrBadRequest error = errors.New("bad request")
	ErrCompiler   error = errors.New("compiler")
	ErrInternal   error = errors.New("internal")
	ErrProgram    error = errors.New("program")
)
