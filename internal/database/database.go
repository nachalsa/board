package database

import (
	"database/sql"
	"log"

	"file-board/internal/config"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func New(cfg *config.Config) (*DB, error) {
	conn, err := sql.Open("postgres", cfg.GetDatabaseURL())
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	log.Println("데이터베이스 연결 성공")
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) GetConnection() *sql.DB {
	return db.conn
}
