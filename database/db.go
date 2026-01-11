package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/rehiy/web-modem/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/glebarez/sqlite"
)

var (
	db     *gorm.DB
	once   sync.Once
	dbPath string
)

// InitDB 初始化数据库连接
func InitDB() error {
	var err error
	once.Do(func() {
		// 获取数据库路径
		dbPath = os.Getenv("DB_PATH")
		if dbPath == "" {
			homeDir, _ := os.UserHomeDir()
			dbPath = filepath.Join(homeDir, ".web-modem", "data.db")
		}

		// 创建目录
		dir := filepath.Dir(dbPath)
		if err = os.MkdirAll(dir, 0755); err != nil {
			return
		}

		// 连接数据库
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return
		}

		// 设置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(2)

		// 创建表
		if err = createTables(); err != nil {
			return
		}

		log.Printf("Database initialized at: %s", dbPath)
	})
	return err
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return db
}

// createTables 创建数据表
func createTables() error {
	// 自动迁移
	err := db.AutoMigrate(
		&models.SMS{},
		&models.Webhook{},
		&models.Setting{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// 确保索引存在（GORM 的 AutoMigrate 会根据标签创建索引，但为了保险，我们手动创建）
	// 如果索引不存在，手动创建（SQLite 不支持 CREATE INDEX IF NOT EXISTS，但我们可以忽略错误）
	// 这里我们依赖 GORM 的标签，不额外创建。

	// 插入默认设置
	defaultSettings := map[string]string{
		"smsdb_enabled": "true",
		"webhook_enabled": "false",
	}

	for key, value := range defaultSettings {
		setting := models.Setting{Key: key, Value: value}
		result := db.FirstOrCreate(&setting, models.Setting{Key: key})
		if result.Error != nil {
			return fmt.Errorf("failed to insert default setting: %w", result.Error)
		}
	}

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
