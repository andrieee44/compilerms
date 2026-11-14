# shlite

## NAME

shlite - shell-free wrapper with globbing and env filtering

## SYNOPSIS

```shell
nix run github:andrieee44/compilerms#shlite -- <ENV_KEY>[:<ENV_KEY>...] <PROGRAM> [ARG]...
```

```shell
shlite <ENV_KEY>[:<ENV_KEY>...] <PROGRAM> [ARG]...
```

## DESCRIPTION

`shlite` is a wrapper that performs glob expansion and environment variable
filtering before executing a program. It allows programs such as `gcc` to be
invoked with glob patterns without relying on a system shell.

The first argument is a colon-separated list of environment variable names.
Only the listed variables are passed through to the executed program. All
others are removed from its environment. The remaining arguments specify the
program to run and its arguments, with glob patterns expanded by `shlite`
itself.

This was custom made for the
[compilerms](https://github.com/andrieee44/compilerms) compiler microservice.

## EXAMPLES

```shell
nix run github:andrieee44/compilerms#shlite -- PATH echo *
```

```shell
shlite PATH echo \*
```

```shell
shlite PATH:HOME:PWD env
```

## COPYRIGHT

See [`LICENSE`](./../../LICENSE). Uses
[AGPLv3 or later](https://www.gnu.org/licenses/agpl-3.0.html).

## SEE ALSO

- [GitHub repository](https://github.com/andrieee44/compilerms)
- [Nix flakes](https://wiki.nixos.org/wiki/Flakes)
- [Nix](https://nixos.org/)
- [Rust](https://www.rust-lang.org/)
- [compilerms](./../../README.md)
