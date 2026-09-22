# compilerms

## NAME

compilerms - Compiler Microservice

## SYNOPSIS

```shell
nix run github:andrieee44/compilerms -- <ADDRESS>
```

```shell
compilerms <ADDRESS>
```

### POST /gcc

`POST /gcc` compiles and runs C source code using [GCC](https://gcc.gnu.org/),
with an opinionated set of warning flags,
[UBSan](https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html)
instrumentation, and static analysis (`-fanalyzer`) enabled.

```haskell
POST /gcc :: {
  headers :: AttrSet;
  sources :: AttrSet;
  stdin :: String;
} -> {
  output :: String;
  status :: Int;
  reason :: String;
}
```

### POST /java

`POST /java` compiles and runs Java source code using `javac`, with linting
(`-Xlint`) enabled. The `Main` class is assumed to contain the program's
entry point.

```haskell
POST /java :: {
  sources :: AttrSet;
  stdin :: String;
} -> {
  output :: String;
  status :: Int;
  reason :: String;
}
```

### POST /sqlite3tmp

`POST /sqlite3tmp` runs a SQL script against a fresh, temporary, in-memory
[SQLite](https://sqlite.org/) database via
[sqlite3tmp](./modules/sqlite3tmp/README.md).

```haskell
POST /sqlite3tmp :: {
  stdin :: String;
} -> {
  output :: String;
  status :: Int;
  reason :: String;
}
```

## DESCRIPTION

compilerms is an opinionated compiler microservice, originally built for a
university server that permits none of [Docker](https://www.docker.com/),
[KVM](https://www.linux-kvm.org/), [systemd](https://systemd.io/) containers,
or [cgroup](https://man7.org/linux/man-pages/man7/cgroups.7.html) creation.
Because of that, its sandboxing architecture is assembled from several
lower-privilege mechanisms rather than a single container runtime:

- [`systemd-run`](https://www.freedesktop.org/software/systemd/man/latest/systemd-run.html)
  provides scope-level resource limits (CPU, memory, tasks) without
  requiring cgroup creation privileges.
- [`ulimit`](https://man7.org/linux/man-pages/man1/ulimit.1p.html) enforces
  process-level resource limits as a second line of defense.
- [`bwrap`](https://github.com/containers/bubblewrap) (bubblewrap) provides
  namespace isolation and a restricted filesystem jail without root or a
  container runtime.
- [seccomp-bpf](https://man7.org/linux/man-pages/man2/seccomp.2.html) filters
  the syscalls available to the sandboxed process, blocking anything
  unnecessary.
- [`trapseccomp`](./modules/trapseccomp/README.md) wraps the sandboxed process
  via [`ptrace(2)`](https://man7.org/linux/man-pages/man2/ptrace.2.html) so
  that seccomp violations (`SIGSYS`) are reported without needing root
  access to read `dmesg`, making it clear exactly which syscall caused a
  compile or program run to be killed.

This is a fairly manual approach, built around what is actually possible on
a restricted server: no root access, no Docker, no KVM, no cgroups. Because
this setup targets a niche of a niche, there is currently no plan to expose
configuration for tweaking flags, such as disabling warnings or adding your
own. If you want different behavior, fork the code (note: this is an
[AGPL](https://www.gnu.org/licenses/agpl-3.0.html) project). If you do have
elevated privileges on your host, better options exist, such as running this
inside a [virtual machine](https://en.wikipedia.org/wiki/Virtual_machine).
Think carefully before self-hosting this microservice directly.

Every endpoint returns the same shape of response: `output` is stdout and
stderr combined, `status` is the process's exit status code, and `reason`
describes which stage of the process was killed. Currently there are only
two possible reasons, `"program error"` and `"compiler error"`, though more
may be added in the future.

## EXAMPLES

```shell
nix run github:andrieee44/compilerms -- localhost:8080
```

```shell
compilerms localhost:8080
```

## COPYRIGHT

See [`LICENSE`](./LICENSE). Uses
[AGPLv3 or later](https://www.gnu.org/licenses/agpl-3.0.html).

## SEE ALSO

- [GCC](https://gcc.gnu.org/)
- [GitHub repository](https://github.com/andrieee44/compilerms)
- [Java](https://openjdk.org/)
- [Nix flakes](https://wiki.nixos.org/wiki/Flakes)
- [Nix](https://nixos.org/)
- [SQLite](https://sqlite.org/)
- [bubblewrap (bwrap)](https://github.com/containers/bubblewrap)
- [seccomp(2)](https://man7.org/linux/man-pages/man2/seccomp.2.html)
- [sqlite3tmp](./modules/sqlite3tmp/README.md)
- [trapseccomp](./modules/trapseccomp/README.md)
