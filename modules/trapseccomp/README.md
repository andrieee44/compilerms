# trapseccomp

## NAME

trapseccomp - Seccomp violation trapping wrapper

## SYNOPSIS

```shell
nix run github:andrieee44/compilerms#trapseccomp -- <BINARY> [ARG]...
```

```shell
trapseccomp <BINARY> [ARG]...
```

## DESCRIPTION

trapseccomp is a
[ptrace(2)](https://man7.org/linux/man-pages/man2/ptrace.2.html)-based wrapper
that reports [seccomp](https://man7.org/linux/man-pages/man2/seccomp.2.html)
violations (`SIGSYS`) from the process it launches, so you don't need root
access to read `dmesg` just to find out which syscall got your program killed.

trapseccomp does not install a seccomp filter itself. It's meant to be run in
conjunction with a program that does, such as
[bubblewrap](https://github.com/containers/bubblewrap) (`bwrap`) with a
`--seccomp` filter file. For violations to be caught, the BPF filter's action
must be `SECCOMP_RET_TRAP` because a `SECCOMP_RET_KILL_PROCESS` or
`SECCOMP_RET_KILL_THREAD` action terminates the tracee directly and bypasses
this wrapper entirely. See
[seccomp(2)](https://man7.org/linux/man-pages/man2/seccomp.2.html) for details
on these filter return actions.

This tool was built to complement the
[compilerms](https://github.com/andrieee44/compilerms) compiler microservice,
which uses it to surface exactly why a sandboxed compile or program run was
killed.

## EXAMPLES

```shell
nix run github:andrieee44/compilerms#trapseccomp -- echo hello world
```

```shell
trapseccomp echo hello world
```

The example above won't show any seccomp violations, since `echo` has no
seccomp filter applied to it and runs unrestricted.

For a concrete example that pairs trapseccomp with a real seccomp filter, see
[gcc.go](./../compilerms/_src/compilerms/compilers/gcc.go), which runs
trapseccomp wrapping `bwrap` to load the filter before executing GCC and the
compiled program.

## COPYRIGHT

See [`LICENSE`](./../../LICENSE). Uses
[AGPLv3 or later](https://www.gnu.org/licenses/agpl-3.0.html).

## SEE ALSO

- [GitHub repository](https://github.com/andrieee44/compilerms)
- [Nix flakes](https://wiki.nixos.org/wiki/Flakes)
- [Nix](https://nixos.org/)
- [bubblewrap (bwrap)](https://github.com/containers/bubblewrap)
- [compilerms](./../../README.md)
- [ptrace(2)](https://man7.org/linux/man-pages/man2/ptrace.2.html)
- [seccomp(2)](https://man7.org/linux/man-pages/man2/seccomp.2.html)
