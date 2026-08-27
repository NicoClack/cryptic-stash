package entps

import (
	"database/sql"
	"database/sql/driver"

	"modernc.org/sqlite"
)

type sqlite3Driver struct {
	*sqlite.Driver
}

type sqlite3DriverConn interface {
	//nolint:inamedparam
	Exec(string, []driver.Value) (driver.Result, error)
}

//nolint:nonamedreturns
func (d sqlite3Driver) Open(name string) (conn driver.Conn, stdErr error) {
	conn, stdErr = d.Driver.Open(name)
	if stdErr != nil {
		return
	}
	_, stdErr = conn.(sqlite3DriverConn).Exec("PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 500;", nil)
	if stdErr != nil {
		_ = conn.Close()
		return
	}
	return
}

func init() {
	sql.Register("sqlite3", sqlite3Driver{Driver: &sqlite.Driver{}})
}
