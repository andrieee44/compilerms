package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	sqlite "gosqlite.org"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	pragmas sqlite.Pragmas = sqlite.Pragmas{
		ForeignKeys: true,
	}

	limits map[int]int = map[int]int{
		sqlite.SQLITE_LIMIT_LENGTH:              1000000,
		sqlite.SQLITE_LIMIT_SQL_LENGTH:          100000,
		sqlite.SQLITE_LIMIT_COLUMN:              100,
		sqlite.SQLITE_LIMIT_EXPR_DEPTH:          10,
		sqlite3.SQLITE_LIMIT_PARSER_DEPTH:       100,
		sqlite.SQLITE_LIMIT_COMPOUND_SELECT:     3,
		sqlite.SQLITE_LIMIT_VDBE_OP:             25000,
		sqlite.SQLITE_LIMIT_FUNCTION_ARG:        8,
		sqlite.SQLITE_LIMIT_ATTACHED:            0,
		sqlite.SQLITE_LIMIT_LIKE_PATTERN_LENGTH: 50,
		sqlite.SQLITE_LIMIT_VARIABLE_NUMBER:     10,
		sqlite.SQLITE_LIMIT_TRIGGER_DEPTH:       10,
		sqlite.SQLITE_LIMIT_WORKER_THREADS:      1,
	}

	dbConfigs map[sqlite.DBConfigOp]bool = map[sqlite.DBConfigOp]bool{
		sqlite.DBConfigDefensive:     true,
		sqlite.DBConfigDQSDML:        false,
		sqlite.DBConfigDQSDDL:        false,
		sqlite.DBConfigTrustedSchema: false,
	}
)

func authorizer(op int, _, _, _, _ string) int {
	switch op {
	case sqlite.SQLITE_PRAGMA,
		sqlite.SQLITE_ATTACH,
		sqlite.SQLITE_DETACH,
		sqlite.SQLITE_CREATE_VTABLE,
		sqlite.SQLITE_DROP_VTABLE:
		return sqlite.SQLITE_DENY
	default:
		return sqlite.SQLITE_OK
	}
}

func run() error {
	var (
		db   *sqlite.DB
		conn *sql.Conn
		err  error
	)

	db, err = sqlite.Open(sqlite.Config{
		Path:            sqlite.InMemory,
		Mode:            sqlite.ModeReadWrite,
		Pragmas:         pragmas,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 0,
	})
	if err != nil {
		return err
	}

	defer db.Close() //nolint:errcheck

	conn, err = db.Conn(context.Background())
	if err != nil {
		return err
	}

	defer conn.Close() //nolint:errcheck

	err = conn.Raw(func(driverConn any) error {
		var (
			c                 *sqlite.Conn
			dbConfigOp        sqlite.DBConfigOp
			enable            bool
			limitID, limitVal int
			err               error
		)

		c = driverConn.(*sqlite.Conn)
		c.RegisterAuthorizer(authorizer)

		err = c.EnableLoadExtension(false)
		if err != nil {
			return err
		}

		for dbConfigOp, enable = range dbConfigs {
			_, err = c.SetDBConfig(dbConfigOp, enable)
			if err != nil {
				return err
			}
		}

		for limitID, limitVal = range limits {
			c.SetLimit(limitID, limitVal)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, `sqlite3tmp: %v

Usage: sqlite3tmp
`, err)

		os.Exit(1)
	}
}
