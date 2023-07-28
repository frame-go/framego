package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

const gormConnMaxIdle = 1 * time.Minute
const gormConnMaxLife = 10 * time.Minute
const gormMaxIdleConns = 10
const gormMaxOpenConns = 0 // Unlimited
const gormSlowQueryThreshold = 200 * time.Millisecond
const gormDsn = "%s:%s@(%s)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&timeout=10s&interpolateParams=true&parseTime=true"

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
	tpl := fmt.Sprintf(gormDsn, c.config.User, c.config.Password, "%s", c.config.Database)
	sources := make([]gorm.Dialector, len(c.config.Masters))
	for i, address := range c.config.Masters {
		sources[i] = mysql.Open(fmt.Sprintf(tpl, address))
	}
	replicas := make([]gorm.Dialector, len(c.config.Slaves))
	for i, address := range c.config.Slaves {
		replicas[i] = mysql.Open(fmt.Sprintf(tpl, address))
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
