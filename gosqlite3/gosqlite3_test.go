package gosqlite3_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/mattn/go-sqlite3"
	"github.com/xaviatwork/gosqlite3/gosqlite3"
)

func TestConnect(t *testing.T) {
	type testCase struct {
		description string
		input       string
		output      error
	}
	testcase := []testCase{
		{description: "connection succeeds", input: "file:db4test.db", output: nil},
		{description: "connection fails", input: "file:/root/db4test.db", output: sqlite3.ErrCantOpen},
	}
	for _, tc := range testcase {
		_, err := gosqlite3.Connect(tc.input)
		if err != nil {
			if sqlite3Err := err.(sqlite3.Error); sqlite3Err.Code != tc.output {
				t.Errorf("%s (for %q): %s", tc.description, tc.input, err.Error())
			}
		}
	}
}

func TestAdd(t *testing.T) {
	dsn := "file:db4test.db"
	db, email := setupDB(dsn, t)
	u := &gosqlite3.User{
		Email:    gofakeit.Email(),
		Password: gofakeit.Password(true, true, true, true, false, 15),
	}
	if err := db.Add(u); err != nil {
		t.Errorf("failed to insert user: %s", err.Error())
	}
	t.Logf("(add) added email: %s", u.Email)

	t.Cleanup(func() {
		db.Delete(email) // Delete user created by setupDB
		t.Logf("(cleanup) deleted user %s", email)
		db.Delete(u.Email) // Delete user created by TestAdd
		t.Logf("(cleanup) deleted user %s", u.Email)
	})
}

func TestDelete(t *testing.T) {
	dsn := "file:db4test.db"
	db, email := setupDB(dsn, t)
	t.Logf("(delete): test email: %s", email)
	if err := db.Delete(email); err != nil {
		t.Errorf("failed to delete user %s, %s", email, err.Error())
	}
}

func TestGet(t *testing.T) {
	type testCase struct {
		description string
		input       string
		output      error
	}
	dsn := "file:db4test.db"

	db, email := setupDB(dsn, t)
	testcase := []testCase{
		{description: "existing email", input: email, output: nil},
		{description: "email not found", input: "non-existing@mail.net", output: sql.ErrNoRows},
	}

	for _, tc := range testcase {
		t.Logf("(get) email: %s", tc.input)

		u, err := db.Get(tc.input)
		if !errors.Is(err, tc.output) {
			t.Errorf("failed to get user %s: %s", email, err.Error())
			continue
		}

		if err == nil && u.Email != tc.input {
			t.Errorf("error retrieving user; got %s but wanted %s", u.Email, tc.input)
		}

		t.Cleanup(func() {
			db.Delete(email)
			t.Logf("(cleanup) deleted user %s", u.Email)
		})
	}
}

func TestUpdate(t *testing.T) {
	dsn := "file:db4test.db"
	db, email := setupDB(dsn, t)
	t.Logf("(update) email: %s", email)
	u, err := db.Get(email)
	if err != nil {
		t.Errorf("failed to get user %s, %s", email, err.Error())
	}

	u.Password = "updated_p@55w0rD"
	if err := db.Update(u); err != nil {
		t.Errorf("error updating user %s: %s", u.Email, err.Error())
	}

	t.Cleanup(func() {
		db.Delete(u.Email)
		t.Logf("(cleanup) deleted user %s", u.Email)
	})
}

func setupDB(dsn string, t *testing.T) (*gosqlite3.Database, string) {
	db, err := gosqlite3.Connect(dsn)
	if err != nil {
		t.Errorf("db setup failed: %s", err.Error())
	}
	u := &gosqlite3.User{
		Email:    gofakeit.Email(),
		Password: gofakeit.Password(true, true, true, true, false, 15),
	}
	err = db.Add(u)
	if err != nil {
		t.Errorf("db setup failed: insert user: %s", err.Error())
	}
	t.Logf("(setupDB) test email: %s", u.Email)
	return db, u.Email
}

func Test_setupDB(t *testing.T) {
	dsn := "file:db4test.db"
	db, email := setupDB(dsn, t)
	if db == nil || email == "" {
		t.Errorf("db setup failed with no error")
	}
	t.Cleanup(func() {
		db.Delete(email) // Delete user created by Test_setupDB
		t.Logf("(cleanup) deleted user %s", email)
	})
}
