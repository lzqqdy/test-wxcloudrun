package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbMu sync.Mutex
	db   *sql.DB
)

func mysqlAddress() string {
	if v := os.Getenv("MYSQL_ADDRESS"); v != "" {
		return v
	}
	return "10.35.108.30:3306"
}

func mysqlUser() string {
	if v := os.Getenv("MYSQL_USERNAME"); v != "" {
		return v
	}
	return "root"
}

func mysqlDatabase() string {
	if v := os.Getenv("MYSQL_DATABASE"); v != "" {
		return v
	}
	return "testdb"
}

func mysqlPassword() string {
	return os.Getenv("MYSQL_PASSWORD")
}

func mysqlDSN(database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=Local&timeout=10s&readTimeout=15s&writeTimeout=15s",
		mysqlUser(), mysqlPassword(), mysqlAddress(), database)
}

func initDB() error {
	if mysqlPassword() == "" {
		return fmt.Errorf("MYSQL_PASSWORD 为空：在云托管「服务设置 → 环境变量」里配置，或确认同环境 MySQL 已自动注入")
	}

	rootDSN := mysqlDSN("")
	rootDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return err
	}
	defer rootDB.Close()
	rootDB.SetConnMaxLifetime(time.Minute)

	if err := pingWithRetry(rootDB, 6); err != nil {
		return fmt.Errorf("连接 MySQL %s 失败（库可能自动暂停，稍后重试）: %w", mysqlAddress(), err)
	}

	dbName := mysqlDatabase()
	if _, err := rootDB.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` DEFAULT CHARACTER SET utf8"); err != nil {
		return fmt.Errorf("创建数据库: %w", err)
	}

	conn, err := sql.Open("mysql", mysqlDSN(dbName))
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(time.Hour)

	if err := pingWithRetry(conn, 5); err != nil {
		_ = conn.Close()
		return err
	}

	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id INT NOT NULL AUTO_INCREMENT,
			text VARCHAR(500) NOT NULL,
			openid VARCHAR(64) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8
	`); err != nil {
		_ = conn.Close()
		return fmt.Errorf("创建表: %w", err)
	}

	dbMu.Lock()
	if db != nil {
		_ = db.Close()
	}
	db = conn
	dbMu.Unlock()
	log.Printf("MySQL 已连接 %s / %s", mysqlAddress(), dbName)
	return nil
}

func pingWithRetry(conn *sql.DB, attempts int) error {
	var err error
	for i := 1; i <= attempts; i++ {
		err = conn.Ping()
		if err == nil {
			return nil
		}
		log.Printf("MySQL ping 第 %d/%d 次失败: %v", i, attempts, err)
		time.Sleep(2 * time.Second)
	}
	return err
}

func ensureDB() error {
	dbMu.Lock()
	conn := db
	dbMu.Unlock()
	if conn != nil {
		if err := conn.Ping(); err == nil {
			return nil
		}
		log.Printf("MySQL 连接失效，尝试重连")
	}
	return initDB()
}

func dbStatus() map[string]any {
	info := map[string]any{
		"ok":       false,
		"address":  mysqlAddress(),
		"user":     mysqlUser(),
		"database": mysqlDatabase(),
		"password": "unset",
	}
	if mysqlPassword() != "" {
		info["password"] = "set"
	}
	if err := ensureDB(); err != nil {
		info["error"] = err.Error()
		return info
	}
	info["ok"] = true
	return info
}
