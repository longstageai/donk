package sql

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	return &DB{db}, nil
}

func (d *DB) Close() error {
	return d.DB.Close()
}

func (d *DB) ExecSchemas(sqls ...string) error {
	for _, s := range sqls {
		if _, err := d.DB.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) MustExec(sqls ...string) {
	for _, s := range sqls {
		if _, err := d.DB.Exec(s); err != nil {
			panic(fmt.Sprintf("执行 SQL 失败: %v", err))
		}
	}
}
