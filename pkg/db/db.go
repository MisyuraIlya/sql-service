package db

import (
	"database/sql"
	"fmt"
	"sql-service/configs"

	_ "github.com/denisenkom/go-mssqldb"
)

type Db struct {
	*sql.DB
}

func NewConnection(cfg *configs.Config) (*Db, error) {
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
		cfg.DbConfig.Server, cfg.DbConfig.User, cfg.DbConfig.Password, cfg.DbConfig.Port, cfg.DbConfig.Database)

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Db{db}, nil
}

func (db *Db) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}
