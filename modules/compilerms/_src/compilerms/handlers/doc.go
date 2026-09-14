// Package handlers provides HTTP handlers that expose the compilers
// package's sandboxed compile-and-run functionality over HTTP.
//
// It exists mainly to keep main lean: request parsing, response encoding,
// and handler logic live here, while routing is wired up in main and the
// actual compiling and running of untrusted code is delegated to the
// compilers package.
package handlers
