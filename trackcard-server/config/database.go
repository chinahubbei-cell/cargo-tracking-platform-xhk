package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func resolveSQLiteDBPath() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("DB_PATH")); explicit != "" {
		return filepath.Abs(explicit)
	}
	if explicit := strings.TrimSpace(os.Getenv("SQLITE_DB_PATH")); explicit != "" {
		return filepath.Abs(explicit)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	candidates := make([]string, 0, 6)
	if filepath.Base(cwd) == "trackcard-server" {
		candidates = append(candidates, filepath.Join(cwd, "trackcard.db"))
	} else {
		candidates = append(candidates, filepath.Join(cwd, "trackcard-server", "trackcard.db"))
		candidates = append(candidates, filepath.Join(cwd, "trackcard.db"))
	}

	if exePath, exeErr := os.Executable(); exeErr == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "trackcard.db"),
			filepath.Join(exeDir, "trackcard-server", "trackcard.db"),
			filepath.Join(exeDir, "..", "trackcard-server", "trackcard.db"),
		)
	}

	seen := make(map[string]struct{}, len(candidates))
	ordered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		ordered = append(ordered, clean)
	}

	for _, candidate := range ordered {
		if _, statErr := os.Stat(candidate); statErr == nil {
			return filepath.Abs(candidate)
		}
	}

	// 若尚未创建数据库，使用第一候选路径
	return filepath.Abs(ordered[0])
}

func InitDatabase(cfg *Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// 通过环境变量 DB_TYPE 选择数据库类型
	// 默认使用 sqlite（适合开发环境）
	// 生产环境设置 DB_TYPE=postgres
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite" // 默认使用SQLite
	}

	switch dbType {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
		)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		log.Println("📦 PostgreSQL Database connected successfully")

	default: // sqlite
		// 配置 SQLite 支持并发：WAL 模式 + busy_timeout + 性能优化
		// WAL 模式允许并发读写，busy_timeout 设置锁等待时间（毫秒）
		// synchronous=NORMAL: 减少磁盘同步，提升写入速度
		// cache_size=-64000: 64MB页面缓存 (负数表示KB)
		// temp_store=MEMORY: 临时表存储在内存中
		dbFilePath, resolveErr := resolveSQLiteDBPath()
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve SQLite path: %w", resolveErr)
		}
		if mkErr := os.MkdirAll(filepath.Dir(dbFilePath), 0o755); mkErr != nil {
			return nil, fmt.Errorf("failed to create SQLite dir: %w", mkErr)
		}
		dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON&_synchronous=NORMAL&_cache_size=-64000&_temp_store=MEMORY", dbFilePath)
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}
		log.Printf("📦 SQLite Database connected successfully (path=%s, WAL mode + performance optimized)", dbFilePath)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 根据数据库类型设置连接池
	if dbType == "postgres" || dbType == "postgresql" {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	} else {
		// SQLite WAL 模式支持多连接并发读取
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetMaxOpenConns(10)
	}

	DB = db
	return db, nil
}
