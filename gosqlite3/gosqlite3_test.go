package gosqlite3_test

import (
	"testing"

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
		{description: "connection succeeds", input: "file::memory:", output: nil},
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
