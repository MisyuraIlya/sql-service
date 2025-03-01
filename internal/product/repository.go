package product

import (
	"sql-service/pkg/db"
	"sql-service/pkg/redis"
)

type ProductRepository struct {
	Db    *db.Db
	Redis *redis.Redisdb
}

func NewProductRepository(db *db.Db, redis *redis.Redisdb) *ProductRepository {
	return &ProductRepository{
		Db:    db,
		Redis: redis,
	}
}
