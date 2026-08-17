package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	dbMu sync.Mutex
	db   *sql.DB
)

func mysqlAddress() string {
	if v := strings.TrimSpace(os.Getenv("MYSQL_ADDRESS")); v != "" {
		return v
	}
	return "10.35.108.30:3306"
}

func mysqlUser() string {
	if v := strings.TrimSpace(os.Getenv("MYSQL_USERNAME")); v != "" {
		return v
	}
	return "root"
}

func mysqlDatabase() string {
	if v := strings.TrimSpace(os.Getenv("MYSQL_DATABASE")); v != "" {
		return v
	}
	return "testdb"
}

func mysqlPassword() string {
	return strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
}

func mysqlDSN(database string) string {
	cfg := mysql.NewConfig()
	cfg.User = mysqlUser()
	cfg.Passwd = mysqlPassword()
	cfg.Net = "tcp"
	cfg.Addr = mysqlAddress()
	cfg.DBName = database
	cfg.Params = map[string]string{"charset": "utf8", "loc": "Local"}
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 15 * time.Second
	cfg.WriteTimeout = 15 * time.Second
	cfg.AllowNativePasswords = true
	return cfg.FormatDSN()
}

func wrapDBErr(err error) error {
	if err == nil {
		return nil
	}
	if isAccessDenied(err) {
		return fmt.Errorf("MySQL 拒绝了账号 %s（1045）。服务环境变量 MYSQL_PASSWORD 和「MySQL → 账号管理」里 root 的当前密码不一致。到「服务设置 → 环境变量」改成同一份密码，保存后重新发布，不要只改代码", mysqlUser())
	}
	if isResuming(err) {
		return fmt.Errorf("MySQL 正在从自动暂停恢复，请等几秒后点「重新连接」: %w", err)
	}
	return fmt.Errorf("连接 MySQL %s 失败: %w", mysqlAddress(), err)
}

func isAccessDenied(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1045
}

func isResuming(err error) bool {
	return err != nil && strings.Contains(err.Error(), "resuming")
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
		return wrapDBErr(err)
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
		return wrapDBErr(err)
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
		if isAccessDenied(err) {
			return err
		}
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
		"ok":             false,
		"address":        mysqlAddress(),
		"user":           mysqlUser(),
		"database":       mysqlDatabase(),
		"password":       "unset",
		"passwordLength": 0,
	}
	if n := len(mysqlPassword()); n > 0 {
		info["password"] = "set"
		info["passwordLength"] = n
	}
	if err := ensureDB(); err != nil {
		info["error"] = err.Error()
		return info
	}
	info["ok"] = true
	return info
}
