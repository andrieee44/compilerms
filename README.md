# compilerms

## NAME

compilerms - a compiler microservice

## SYNOPSIS

```shell
nix run github:andrieee44/compilerms -- <CONFIG_FILE>
```

```shell
compilerms <CONFIG_FILE>
```

### GET /

`GET /` returns the available languages and the versions of their components.

```go
GET / -> struct { versions map[string]map[string]string }
```

### GET /{language}

`GET /{language}` is live mode. After the WebSocket connection is established,
the client first sends a JSON message containing the files, and then exchanges
binary messages for input and output. The server returns the program output as
binary messages, followed by a JSON message containing the version(s) of the
program(s) used, its exit status, and the name of the program.

```go
GET /{language} :: struct { files map[string]string } -> struct {
  versions map[string]string
  name     string
  status   int
}
```

### POST /{language}

`POST /{language}` is batch mode, which simplifies execution into a single
request and response.

```go
POST /{language} :: struct {
  files map[string]string;
  input string;
} -> struct {
  versions map[string]string
  name     string
  output   string
  status   int
}
```

## DESCRIPTION

compilerms is a compiler microservice that supports both batch and live modes.
It is configured through a [Jsonnet](https://jsonnet.org/) file, chosen because
it allows configuration to be programmed, which is not possible with formats
such as TOML, YAML, or plain JSON. Batch requests use the POST endpoint, live
requests use the GET endpoint, and a GET request to the root path returns the
available languages and their versions.

As a best practice, sandboxing tools such as
[bubblewrap](https://github.com/containers/bubblewrap),
[systemd-run](https://www.freedesktop.org/software/systemd/man/latest/systemd-run.html),
and [ulimit](https://man7.org/linux/man-pages/man3/ulimit.3.html) should be
used. [shlite](./modules/shlite/README.md) is provided so that glob patterns,
such as those passed as arguments to gcc, can be expanded without a full shell.
It also clears all environment variables except those explicitly permitted.

compilerms provides a temporary directory that is generated on the fly. You
can access it by templating your arguments with `{{ COMPILERMS_RUNDIR }}`,
which is meant for storing compiled programs, artifacts, and other files, and
`{{ COMPILERMS_RODIR }}`, which is populated by compilerms to contain the files
passed by the client. Both directories are backed by the same temporary
directory and are named `{tmpDir}/rundir` and `{tmpDir}/rodir` respectively.

On startup, compilerms runs a mandatory test suite. Because the locations of
files and executables differ between Linux distributions, this prevents errors
that would otherwise result from copying a configuration verbatim. See
[EXAMPLES](#examples) for more details.

The supported configuration options are defined in
[config.go](./modules/compilerms/_src/config.go).

## EXAMPLES

Examples of how to call each endpoint can be found in
[test.go](./modules/compilerms/_src/test.go).

A concrete example that uses systemd-run and shlite for sandboxing is provided
in the [configuration directory](./modules/compilerms/_src/config), which
currently supports only gcc and bash. Within that directory,
[compilerms.libsonnet](./modules/compilerms/_src/config/compilerms.libsonnet)
is a helper library that generates the debug and production arguments,
wrapping each command in systemd-run, strace, and shlite.

`shlite` is a custom program, documented in its
[README.md](./modules/shlite/README.md), that performs shell-style globbing and
removes environment variables.

```shell
nix run github:andrieee44/compilerms -- /path/to/config.jsonnet
```

```shell
compilerms /path/to/config.jsonnet
```

## COPYRIGHT

See [`LICENSE`](./LICENSE). Uses
[AGPLv3 or later](https://www.gnu.org/licenses/agpl-3.0.html).

## SEE ALSO

- [GCC](https://gcc.gnu.org/)
- [GitHub repository](https://github.com/andrieee44/compilerms)
- [Go](https://go.dev/)
- [Jsonnet](https://jsonnet.org/)
- [Nix flakes](https://wiki.nixos.org/wiki/Flakes)
- [Nix](https://nixos.org/)
- [WebSocket](https://datatracker.ietf.org/doc/html/rfc6455)
- [bash](https://www.gnu.org/software/bash/)
- [bubblewrap](https://github.com/containers/bubblewrap)
- [shlite](./modules/shlite/README.md)
- [strace](https://strace.io/)
- [systemd-run](https://www.freedesktop.org/software/systemd/man/latest/systemd-run.html)
