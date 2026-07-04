package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

const gormConnMaxIdle = 1 * time.Minute
const gormConnMaxLife = 10 * time.Minute
const gormMaxIdleConns = 10
const gormMaxOpenConns = 0 // Unlimited
const gormSlowQueryThreshold = 200 * time.Millisecond
const mysqlDsn = "%s:%s@(%s)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&timeout=10s&interpolateParams=true&parseTime=true"

type GormClient struct {
	config  *Config
	options options
	db      *gorm.DB
}

func (c *GormClient) init(config *Config, opts ...Option) error {
	c.config = config
	for _, opt := range opts {
		opt(&c.options)
	}
	sources, err := c.dialectors(c.config.Masters)
	if err != nil {
		return err
	}
	replicas, err := c.dialectors(c.config.Slaves)
	if err != nil {
		return err
	}
	logger := NewGormLogger(c.options.logger)
	logger.SlowThreshold = gormSlowQueryThreshold
	db, err := gorm.Open(
		sources[0],
		&gorm.Config{
			DisableAutomaticPing: true,
			Logger:               logger,
		})
	if err != nil {
		return err
	}
	err = db.Use(
		dbresolver.Register(
			dbresolver.Config{
				Sources:  sources,
				Replicas: replicas,
				Policy:   dbresolver.RandomPolicy{},
			}).
			SetConnMaxIdleTime(gormConnMaxIdle).
			SetConnMaxLifetime(gormConnMaxLife).
			SetMaxIdleConns(gormMaxIdleConns).
			SetMaxOpenConns(gormMaxOpenConns))
	if err != nil {
		return err
	}
	c.db = db
	return nil
}

func (c *GormClient) dialectors(addresses []string) ([]gorm.Dialector, error) {
	ds := make([]gorm.Dialector, len(addresses))
	for i, address := range addresses {
		d, err := c.dialector(address)
		if err != nil {
			return nil, err
		}
		ds[i] = d
	}
	return ds, nil
}

func (c *GormClient) dialector(address string) (gorm.Dialector, error) {
	switch c.config.Type {
	case "", TypeMySQL:
		dsn := fmt.Sprintf(mysqlDsn, c.config.User, c.config.Password, address, c.config.Database)
		return mysql.Open(dsn), nil
	case TypePostgres:
		db, err := openPostgresDB(c.config, address)
		if err != nil {
			return nil, err
		}
		return postgres.New(postgres.Config{Conn: db}), nil
	default:
		return nil, fmt.Errorf("unsupported database type %q", c.config.Type)
	}
}

func (c *GormClient) DB() *gorm.DB {
	return c.db
}

func NewGormClient(config *Config, opts ...Option) (*gorm.DB, error) {
	c := GormClient{}
	err := c.init(config, opts...)
	if err != nil {
		return nil, err
	}
	return c.DB(), nil
}
