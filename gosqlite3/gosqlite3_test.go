package gosqlite3_test

import (
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

func setupDB(dsn string, t *testing.T) (*gosqlite3.Database, string) {
	db, err := gosqlite3.Connect(dsn)
	if err != nil {
		t.Errorf("db setup failed: %s", err.Error())
	}
	u := &gosqlite3.User{
		Email:    gofakeit.Email(),
		Password: gofakeit.Password(true, true, true, true, false, 15),
	}
	return db, u.Email
}

func Test_setupDB(t *testing.T) {
	dsn := "file:db4test.db"
	db, email := setupDB(dsn, t)
	if db == nil || email == "" {
		t.Errorf("db setup failed with no error")
	}
}
