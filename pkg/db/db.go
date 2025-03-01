package db

import "sql-service/configs"

type Db struct {
}

func NewDb(conf *configs.Config) *Db {
	// db, err := gorm.Open(postgres.Open(conf.Db.Dsn), &gorm.Config{})
	// if err != nil {
	// 	panic(err)
	// }
	return &Db{
		// db
	}
}
