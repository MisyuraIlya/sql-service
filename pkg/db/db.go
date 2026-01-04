package db

import (
	"database/sql"
	"fmt"
	"strings"

	"sql-service/configs"

	_ "github.com/SAP/go-hdb/driver"
	_ "github.com/denisenkom/go-mssqldb"
)

type Db struct {
	*sql.DB
	Dialect string
}

func NewConnection(cfg *configs.Config) (*Db, error) {
	dialect := strings.ToLower(strings.TrimSpace(cfg.DbConfig.Dialect))
	if dialect == "" {
		dialect = "mssql"
	}

	var (
		driverName string
		connString string
	)

	if cfg.DbConfig.DSN != "" {
		connString = cfg.DbConfig.DSN
	} else {
		switch dialect {
		case "mssql":
			driverName = "sqlserver"
			connString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
				cfg.DbConfig.Server, cfg.DbConfig.User, cfg.DbConfig.Password, cfg.DbConfig.Port, cfg.DbConfig.Database)
		case "hana":
			driverName = "hdb"
			connString = fmt.Sprintf("hdb://%s:%s@%s:%d?databaseName=%s",
				cfg.DbConfig.User, cfg.DbConfig.Password, cfg.DbConfig.Server, cfg.DbConfig.Port, cfg.DbConfig.Database)
		default:
			return nil, fmt.Errorf("unsupported db dialect: %s", dialect)
		}
	}

	if driverName == "" {
		switch dialect {
		case "mssql":
			driverName = "sqlserver"
		case "hana":
			driverName = "hdb"
		default:
			return nil, fmt.Errorf("unsupported db dialect: %s", dialect)
		}
	}

	db, err := sql.Open(driverName, connString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Db{DB: db, Dialect: dialect}, nil
}

func (db *Db) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}
