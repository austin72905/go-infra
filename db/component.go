package db

import (
    "context"
    "database/sql"
    "fmt"
    "net/url"
    "strconv"
    "time"

    "github.com/austin72905/go-infra/app"
    internalmodule "github.com/austin72905/go-infra/internal/module"
    "gorm.io/gorm"
)

const DefaultComponentName = "DB"

type Component struct {
    appRuntime         *app.Runtime
    client             *Client
    dsn                string
    maxIdleConns       int
    maxOpenConns       int
    connMaxIdleTime    time.Duration
    shutdownRegistered bool
}

func (db *Component) Initialize(runtime *app.Runtime, name string) {
    db.appRuntime = runtime
    if db.shutdownRegistered {
        return
    }

    runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
        _ = db.Close()
    }))
    db.shutdownRegistered = true
}

func (db *Component) Validate() {
}

func (db *Component) SetDSN(dsn string) {
    db.dsn = dsn
    if hostURI := postgresHostFromDSN(dsn); hostURI != "" {
        db.appRuntime.Probe.AddHostURI(hostURI)
    }
}

func (db *Component) SetPoolSize(maxIdleConns, maxOpenConns int) {
    db.maxIdleConns = maxIdleConns
    db.maxOpenConns = maxOpenConns
}

func (db *Component) SetConnMaxIdleTime(d time.Duration) {
    db.connMaxIdleTime = d
}

func (db *Component) LoadFromProperties(dsnKey string) {
    db.SetDSN(db.appRuntime.Property.RequiredProperty(dsnKey))
}

func (db *Component) LoadFromPrefix(prefix string) {
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

func (db *Component) EnsureClient() {
    if db.client != nil {
        return
    }
    if db.dsn == "" {
        panic("db dsn is required")
    }
    db.client = New(db.dsn, db.maxIdleConns, db.maxOpenConns, db.connMaxIdleTime)
}

func (db *Component) Client() *Client {
    db.EnsureClient()
    return db.client
}

func (db *Component) SQLDB() *sql.DB {
    return db.Client().SQLDB()
}

func (db *Component) GormDB() *gorm.DB {
    return db.Client().GormDB()
}

func (db *Component) Ping(ctx context.Context) error {
    return db.Client().Ping(ctx)
}

func (db *Component) MustPing(ctx context.Context) {
    if err := db.Ping(ctx); err != nil {
        panic(fmt.Sprintf("db ping failed: %v", err))
    }
}

func (db *Component) Close() error {
    if db.client == nil {
        return nil
    }
    err := db.client.Close()
    db.client = nil
    return err
}

func ComponentName(name string) string {
    if name == "" {
        return DefaultComponentName
    }
    return DefaultComponentName + ":" + name
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
