package compilers

import "errors"

var (
	ErrCompiler error = errors.New("compiler")
	ErrInternal error = errors.New("internal")
	ErrProgram  error = errors.New("program")
)
