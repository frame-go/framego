package database

import (
	"github.com/linxGnu/mssqlx"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type ClientManager interface {
	GetClient(string) *gorm.DB
	GetGormClient(string) *gorm.DB
	GetSqlxClient(string) *mssqlx.DBs
}

type options struct {
	logger *zerolog.Logger
}

type Option func(*options)

func WithLogger(logger *zerolog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

type clientManagerImpl struct {
	configs     []Config
	opts        []Option
	gormClients map[string]*gorm.DB
	sqlxClients map[string]*mssqlx.DBs
}

func (c *clientManagerImpl) GetClient(name string) *gorm.DB {
	return c.GetGormClient(name)
}

func (c *clientManagerImpl) GetGormClient(name string) *gorm.DB {
	if c.gormClients == nil {
		err := c.initGormClients()
		if err != nil {
			return nil
		}
	}
	client, ok := c.gormClients[name]
	if !ok {
		return nil
	}
	return client
}

func (c *clientManagerImpl) GetSqlxClient(name string) *mssqlx.DBs {
	if c.gormClients == nil {
		err := c.initSqlxClients()
		if err != nil {
			return nil
		}
	}
	client, ok := c.sqlxClients[name]
	if !ok {
		return nil
	}
	return client
}

func (c *clientManagerImpl) initGormClients() error {
	c.gormClients = make(map[string]*gorm.DB)
	for _, config := range c.configs {
		db, err := NewGormClient(&config, c.opts...)
		if err != nil {
			return err
		}
		c.gormClients[config.Name] = db
	}
	return nil
}

func (c *clientManagerImpl) initSqlxClients() error {
	c.sqlxClients = make(map[string]*mssqlx.DBs)
	for _, config := range c.configs {
		db, err := NewSqlxClient(&config, c.opts...)
		if err != nil {
			return err
		}
		c.sqlxClients[config.Name] = db
	}
	return nil
}

func NewClientManager(configs []Config, opts ...Option) (ClientManager, error) {
	c := &clientManagerImpl{
		configs: configs,
		opts:    opts,
	}
	err := c.initGormClients()
	if err != nil {
		return nil, err
	}
	return c, nil
}
