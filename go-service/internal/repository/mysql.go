package repository

import (
	"database/sql"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQL(dsn string)(*sql.DB, error) {
	db, err := sql.Open("mysql",dsn)
	if err != nil {
		return nil,err
	}
	
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil,err
	}

	slog.Info("MySQL connected successfully")
	return db, nil
}
