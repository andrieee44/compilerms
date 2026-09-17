# sqlite3tmp

## NAME

sqlite3tmp - run SQL against an in-memory SQLite database

## SYNOPSIS

```shell
echo SELECT 1 | nix run github:andrieee44/compilerms#sqlite3tmp
```

```shell
echo SELECT 1 | sqlite3tmp
```

## DESCRIPTION

sqlite3tmp is a SQLite command executor with no cgo dependency. It reads an
entire SQL script from stdin, executes each statement against a fresh
in-memory database, and writes the results to stdout as JSON.

Every database is opened with reasonable defaults and hardened following the
guidance in
[SQLite's security documentation](https://sqlite.org/security.html): foreign
keys are enforced, dangerous double-quoted-string quirks and VTABLE/ATTACH
operations are disabled, and query complexity is bounded via `sqlite3_limit()`.

sqlite3tmp does not limit memory usage or wall-clock execution time. If you
need those guarantees, wrap it in another program rather than relying on
sqlite3tmp alone.

This was custom made for the
[compilerms](https://github.com/andrieee44/compilerms) compiler microservice.

## COPYRIGHT

See [`LICENSE`](./../../LICENSE). Uses
[AGPLv3 or later](https://www.gnu.org/licenses/agpl-3.0.html).

## SEE ALSO

- [GitHub repository](https://github.com/andrieee44/compilerms)
- [Nix flakes](https://wiki.nixos.org/wiki/Flakes)
- [Nix](https://nixos.org/)
- [SQLite `sqlite3_limit()`](https://sqlite.org/c3ref/limit.html)
- [SQLite `sqlite3_set_authorizer()`](https://sqlite.org/c3ref/set_authorizer.html)
- [SQLite security guidance](https://sqlite.org/security.html)
- [compilerms](./../../README.md)
- [gosqlite.org](https://gosqlite.org)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)
