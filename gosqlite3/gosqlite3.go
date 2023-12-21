package gosqlite3

import "database/sql"

type Database struct {
	cnx *sql.DB
}
