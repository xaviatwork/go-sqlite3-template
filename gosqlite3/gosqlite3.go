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

var tableName string = "users"

func Connect(dsn string) (*Database, error) {
	driverName := "sqlite3"
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

func (db *Database) Add(u *User) error {
	tx, err := db.cnx.Begin()
	if err != nil {
		return fmt.Errorf("begin 'add' transaction failed: %w", err)
	}

	sqlInsert := fmt.Sprintf("INSERT INTO %s (email, password) VALUES (?,?)", tableName)
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		return fmt.Errorf("prepare 'add' transaction failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.Email, u.Password)
	if err != nil {
		return fmt.Errorf("exec 'add' transaction failed: %w", err)
	}

	tx.Commit()

	return nil
}

func (db *Database) Delete(email string) error {
	tx, err := db.cnx.Begin()
	if err != nil {
		return fmt.Errorf("begin 'delete' transaction failed: %w", err)
	}

	sqlDelete := fmt.Sprintf("DELETE FROM %s WHERE email = ?", tableName)
	stmt, err := tx.Prepare(sqlDelete)
	if err != nil {
		return fmt.Errorf("prepare 'delete' transaction failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(email)
	if err != nil {
		return fmt.Errorf("exec 'delete' transaction failed: %w", err)
	}

	tx.Commit()

	return nil
}

func (db *Database) Get(email string) (*User, error) {
	sqlGet := fmt.Sprintf("SELECT * FROM %s WHERE email = ?", tableName)
	stmt, err := db.cnx.Prepare(sqlGet)
	if err != nil {
		return &User{}, err
	}

	u := &User{}
	err = stmt.QueryRow(email).Scan(&u.Email, &u.Password)
	if err != nil {
		return &User{}, fmt.Errorf("exec 'delete' transaction failed: %w", err)
	}

	return u, nil
}

func (db *Database) Update(u *User) error {
	tx, err := db.cnx.Begin()
	if err != nil {
		return fmt.Errorf("begin 'update' transaction failed: %w", err)
	}

	sqlUpdate := fmt.Sprintf("UPDATE %s SET email = ?, password = ? WHERE email = ?", tableName)
	stmt, err := tx.Prepare(sqlUpdate)
	if err != nil {
		return fmt.Errorf("prepare 'update' transaction failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.Email, u.Password, u.Email)
	if err != nil {
		return fmt.Errorf("exec 'update' transaction failed: %w", err)
	}

	tx.Commit()

	return nil
}
