package database

import (
	"errors"
	"fmt"

	"database/sql"
	"github.com/go-sql-driver/mysql"
	"github.com/linxGnu/mssqlx"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
)

const sqlxDsn = "%s:%s@(%s)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&timeout=10s&interpolateParams=true&parseTime=true"

type MssqlxClient struct {
	config  *Config
	options options
	db      *mssqlx.DBs
}

func (c *MssqlxClient) mysqlInstantiate(driverName, dsn string) (*sql.DB, error) {
	if driverName != "mysql" {
		return nil, errors.New("only_supported_mysql")
	}
	driver := &mysql.MySQLDriver{}
	db := sqldblogger.OpenDriver(
		dsn,
		driver,
		zerologadapter.New(*c.options.logger),
	)
	return db, nil
}

func (c *MssqlxClient) init(config *Config, opts ...Option) error {
	c.config = config
	for _, opt := range opts {
		opt(&c.options)
	}
	tpl := fmt.Sprintf(sqlxDsn, c.config.User, c.config.Password, "%s", c.config.Database)
	masterDsns := make([]string, len(c.config.Masters))
	for i, address := range c.config.Masters {
		masterDsns[i] = fmt.Sprintf(tpl, address)
	}
	slaveDsns := make([]string, len(c.config.Slaves))
	for i, address := range c.config.Slaves {
		slaveDsns[i] = fmt.Sprintf(tpl, address)
	}
	var sqlOptions []mssqlx.Option
	if c.options.logger != nil {
		sqlOptions = append(sqlOptions, mssqlx.WithDBInstantiate(c.mysqlInstantiate))
	}
	db, errs := mssqlx.ConnectMasterSlaves("mysql", masterDsns, slaveDsns, sqlOptions...)
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	c.db = db
	return nil
}

func (c *MssqlxClient) DB() *mssqlx.DBs {
	return c.db
}

func NewSqlxClient(config *Config, opts ...Option) (*mssqlx.DBs, error) {
	c := MssqlxClient{}
	err := c.init(config, opts...)
	if err != nil {
		return nil, err
	}
	return c.DB(), nil
}
