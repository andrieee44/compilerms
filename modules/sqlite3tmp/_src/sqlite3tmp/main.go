package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	sqlite "gosqlite.org"
	sqlite3 "modernc.org/sqlite/lib"
)

type Output struct {
	Columns []string `json:"columns,omitempty"`
	Rows    [][]any  `json:"rows,omitempty"`
}

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
		"sqlite3tmp: blocked operation nr=%d name=%q\n",
		op,
		opStr,
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

func readStatements(rd io.Reader) ([]string, error) {
	var (
		reader     *bufio.Reader
		builder    strings.Builder
		statements []string
		statement  string
		err        error
	)

	reader = bufio.NewReader(rd)

	for {
		for {
			statement, err = reader.ReadString(';')
			if err != nil {
				if errors.Is(err, io.EOF) {
					builder.WriteString(statement)
					builder.WriteByte(';')

					return append(statements, builder.String()), nil
				}

				return nil, err
			}

			builder.WriteString(statement)

			if sqlite.Complete(builder.String()) {
				break
			}
		}

		statements = append(statements, builder.String())
		builder.Reset()
	}
}

func executeStatements(conn *sql.Conn, statements []string) ([]Output, error) {
	var (
		outputs          []Output
		statement        string
		output           Output
		rows             *sql.Rows
		rowPtrs, rowVals []any
		i                int
		err              error
	)

	outputs = make([]Output, 0, len(statements))

	for _, statement = range statements {
		output = Output{}

		rows, err = conn.QueryContext(context.Background(), statement)
		if err != nil {
			return nil, err
		}

		output.Columns, err = rows.Columns()
		if err != nil {
			return nil, err
		}

		if len(output.Columns) == 0 {
			continue
		}

		for rows.Next() {
			rowPtrs = make([]any, len(output.Columns))
			rowVals = make([]any, len(output.Columns))

			for i = range len(output.Columns) {
				rowPtrs[i] = &rowVals[i]
			}

			err = rows.Scan(rowPtrs...)
			if err != nil {
				return nil, err
			}

			output.Rows = append(output.Rows, rowVals)
		}

		err = rows.Err()
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

func run() error {
	var (
		db         *sqlite.DB
		conn       *sql.Conn
		statements []string
		outputs    []Output
		err        error
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

	statements, err = readStatements(os.Stdin)
	if err != nil {
		return err
	}

	outputs, err = executeStatements(conn, statements)
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(outputs)
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
