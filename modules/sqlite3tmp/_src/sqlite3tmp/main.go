package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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
	var opStr string

	switch op {
	case sqlite.SQLITE_PRAGMA:
		opStr = "SQLITE_PRAGMA"
	case sqlite.SQLITE_ATTACH:
		opStr = "SQLITE_ATTACH"
	case sqlite.SQLITE_DETACH:
		opStr = "SQLITE_DETACH"
	case sqlite.SQLITE_CREATE_VTABLE:
		opStr = "SQLITE_CREATE_VTABLE"
	case sqlite.SQLITE_DROP_VTABLE:
		opStr = "SQLITE_DROP_VTABLE"
	default:
		return sqlite.SQLITE_OK
	}

	fmt.Fprintf(
		os.Stderr,
		"sqlite3tmp: blocked operation name=%q nr=%d\n",
		opStr,
		op,
	)

	return sqlite.SQLITE_DENY
}

func rawConnSetup(driverConn any) error {
	var (
		rawConn           *sqlite.Conn
		dbConfigOp        sqlite.DBConfigOp
		dbConfigEnable    bool
		limitID, limitVal int
		err               error
	)

	rawConn = driverConn.(*sqlite.Conn)
	rawConn.RegisterAuthorizer(authorizer)

	for dbConfigOp, dbConfigEnable = range dbConfigs {
		_, err = rawConn.SetDBConfig(dbConfigOp, dbConfigEnable)
		if err != nil {
			return err
		}
	}

	for limitID, limitVal = range limits {
		rawConn.SetLimit(limitID, limitVal)
	}

	return nil
}

func evaluate(conn *sql.Conn, statement string) error {
	var (
		rows             *sql.Rows
		cols             []string
		rowPtrs, rowVals []any
		table            [][]any
		i                int
		err              error
	)

	rows, err = conn.QueryContext(context.Background(), statement)
	if err != nil {
		return err
	}

	cols, err = rows.Columns()
	if err != nil {
		return err
	}

	if len(cols) == 0 {
		return nil
	}

	for rows.Next() {
		rowPtrs = make([]any, len(cols))
		rowVals = make([]any, len(cols))

		for i = range len(cols) {
			rowPtrs[i] = &rowVals[i]
		}

		err = rows.Scan(rowPtrs...)
		if err != nil {
			return err
		}

		table = append(table, rowVals)
	}

	return json.NewEncoder(os.Stdout).Encode(
		struct {
			Cols  []string `json:"cols"`
			Table [][]any  `json:"table"`
		}{
			Cols:  cols,
			Table: table,
		},
	)
}

func read(input []byte) ([]string, []byte) {
	var (
		buf, statement []byte
		statements     []string
	)

	for buf = range bytes.SplitAfterSeq(input, []byte{';'}) {
		statement = append(statement, buf...)

		if sqlite.Complete(string(statement)) {
			statements = append(statements, string(statement))
			statement = []byte{}
		}
	}

	return statements, statement
}

func repl(conn *sql.Conn) error {
	var (
		scanner       *bufio.Scanner
		line          int
		input         []byte
		statements    []string
		statement     string
		firstErr, err error
	)

	scanner = bufio.NewScanner(os.Stdin)
	line = 0

	for scanner.Scan() {
		line++
		input = append(input, scanner.Bytes()...)

		statements, input = read(input)
		for _, statement = range statements {
			err = evaluate(conn, statement)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				err = fmt.Errorf("line %d: %v", line, err)

				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}

	if len(input) != 0 {
		err = evaluate(conn, string(input))
		if err != nil {
			return fmt.Errorf("line %d: %v", line, err)
		}
	}

	if firstErr != nil {
		return firstErr
	}

	return scanner.Err()
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

	conn, err = db.Conn(context.Background())
	if err != nil {
		return err
	}

	err = conn.Raw(rawConnSetup)
	if err != nil {
		return err
	}

	return repl(conn)
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, `sqlite3tmp: %v

Usage: echo SELECT 1 | sqlite3tmp
`, err)

		os.Exit(1)
	}
}
