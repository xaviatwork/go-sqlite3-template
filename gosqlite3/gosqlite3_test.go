package gosqlite3_test

import (
	"testing"

	"github.com/xaviatwork/gosqlite3/gosqlite3"
)

func TestConnect(t *testing.T) {
	dsn := "file::memory:"
	_, err := gosqlite3.Connect(dsn)
	if err != nil {
		t.Errorf("failed to connect to DB %q: %s", dsn, err.Error())
	}
}
