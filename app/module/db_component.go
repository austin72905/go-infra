package module

import (
	"context"
	"database/sql"
	"fmt"
	internaldb "github.com/austin72905/go-infra/internal/db"
	internalmodule "github.com/austin72905/go-infra/internal/module"
	"net/url"
	"strconv"
	"time"

	"gorm.io/gorm"
)

const defaultDBComponentName = "DB"

type DBComponent struct {
	appRuntime         *AppRuntime
	client             *internaldb.Client
	dsn                string // 連線字串
	maxIdleConns       int
	maxOpenConns       int
	connMaxIdleTime    time.Duration
	shutdownRegistered bool
}

func (db *DBComponent) initialize(runtime *AppRuntime, name string) {
	db.appRuntime = runtime
	if db.shutdownRegistered {
		return
	}

	runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
		_ = db.Close()
	}))
	db.shutdownRegistered = true
}

func (db *DBComponent) validate() {
}

func (db *DBComponent) SetDSN(dsn string) {
	db.dsn = dsn
	if hostURI := postgresHostFromDSN(dsn); hostURI != "" {
		db.appRuntime.Probe.AddHostURI(hostURI)
	}
}

func (db *DBComponent) SetPoolSize(maxIdleConns, maxOpenConns int) {
	db.maxIdleConns = maxIdleConns
	db.maxOpenConns = maxOpenConns
}

func (db *DBComponent) SetConnMaxIdleTime(d time.Duration) {
	db.connMaxIdleTime = d
}

func (db *DBComponent) LoadFromProperties(dsnKey string) {
	db.SetDSN(db.appRuntime.Property.RequiredProperty(dsnKey))
}

/*
prefix + ".dsn"
prefix + ".pool.maxIdleConns"
prefix + ".pool.maxOpenConns"
prefix + ".connMaxIdleTime"
可以這樣配置
db.dsn=postgres://user:pass@localhost:5432/app?sslmode=disable
db.pool.maxIdleConns=10
db.pool.maxOpenConns=30
db.connMaxIdleTime=30m

db.slave.dsn=postgres://user:pass@localhost:5432/app_read?sslmode=disable
db.slave.pool.maxIdleConns=10
db.slave.pool.maxOpenConns=30
db.slave.connMaxIdleTime=30m
*/
func (db *DBComponent) LoadFromPrefix(prefix string) {
	if prefix == "" {
		panic("db property prefix is required")
	}

	db.SetDSN(db.appRuntime.Property.RequiredProperty(prefix + ".dsn"))

	maxIdleKey := prefix + ".pool.maxIdleConns"
	maxOpenKey := prefix + ".pool.maxOpenConns"
	maxIdleValue := db.appRuntime.Property.Property(maxIdleKey)
	maxOpenValue := db.appRuntime.Property.Property(maxOpenKey)
	if maxIdleValue != "" || maxOpenValue != "" {
		maxIdleConns, err := strconv.Atoi(defaultString(maxIdleValue, "0"))
		if err != nil {
			panic(fmt.Sprintf("invalid db pool maxIdleConns for key %q: %v", maxIdleKey, err))
		}
		maxOpenConns, err := strconv.Atoi(defaultString(maxOpenValue, "0"))
		if err != nil {
			panic(fmt.Sprintf("invalid db pool maxOpenConns for key %q: %v", maxOpenKey, err))
		}
		db.SetPoolSize(maxIdleConns, maxOpenConns)
	}

	idleTimeKey := prefix + ".connMaxIdleTime"
	idleTimeValue := db.appRuntime.Property.Property(idleTimeKey)
	if idleTimeValue != "" {
		duration, err := time.ParseDuration(idleTimeValue)
		if err != nil {
			panic(fmt.Sprintf("invalid db connMaxIdleTime for key %q: %v", idleTimeKey, err))
		}
		db.SetConnMaxIdleTime(duration)
	}
}

func (db *DBComponent) ensureClient() {
	if db.client != nil {
		return
	}
	if db.dsn == "" {
		panic("db dsn is required")
	}
	db.client = internaldb.New(db.dsn, db.maxIdleConns, db.maxOpenConns, db.connMaxIdleTime)
}

func (db *DBComponent) Client() *internaldb.Client {
	db.ensureClient()
	return db.client
}

func (db *DBComponent) SQLDB() *sql.DB {
	return db.Client().SQLDB()
}

func (db *DBComponent) GormDB() *gorm.DB {
	return db.Client().GormDB()
}

func (db *DBComponent) Ping(ctx context.Context) error {
	return db.Client().Ping(ctx)
}

func (db *DBComponent) MustPing(ctx context.Context) {
	if err := db.Ping(ctx); err != nil {
		panic(fmt.Sprintf("db ping failed: %v", err))
	}
}

func (db *DBComponent) Close() error {
	if db.client == nil {
		return nil
	}
	err := db.client.Close()
	db.client = nil
	return err
}

func dbComponentName(name string) string {
	if name == "" {
		return defaultDBComponentName
	}
	return defaultDBComponentName + ":" + name
}

func postgresHostFromDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	if u.Host == "" {
		return ""
	}
	return u.Host
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
