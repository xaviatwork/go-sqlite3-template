package gosqlite3

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	cnx *sql.DB
}

func Connect(dsn string) (*Database, error) {
	driverName := "sqlite3"
	conn, err := sql.Open(driverName, dsn)
	if err != nil {
		return &Database{}, err
	}
	return &Database{cnx: conn}, nil
}
