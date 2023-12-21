package gosqlite3

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	cnx *sql.DB
}

type User struct {
	Email    string
	Password string
}

func Connect(dsn string) (*Database, error) {
	driverName := "sqlite3"
	tableName := "users"
	sqlCreateTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (email TEXT PRIMARY KEY, password TEXT NOT NULL);", tableName)
	conn, err := sql.Open(driverName, dsn)
	if err != nil {
		return &Database{}, err
	}
	db := &Database{cnx: conn}
	if err := db.cnx.Ping(); err != nil {
		return &Database{}, err
	}
	_, err = db.cnx.Exec(sqlCreateTable)
	if err != nil {
		return &Database{}, err
	}
	return &Database{cnx: conn}, nil
}
