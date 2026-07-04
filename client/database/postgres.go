package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
)

const postgresDsn = "postgres://%s:%s@%s/%s?sslmode=disable&connect_timeout=10"

// openPostgresDB opens a database/sql handle backed by pgx with the uint64 bit-cast
// codec registered, so full-range uint64 values (e.g. snowflake ids) can be bound as
// signed bigint parameters. Reading them back requires a sql.Scanner field type such
// as uniqueid.ID, because result scanning goes through database/sql, which rejects
// negative-to-uint64.
func openPostgresDB(config *Config, address string) (*sql.DB, error) {
	dsn := fmt.Sprintf(postgresDsn,
		url.QueryEscape(config.User), url.QueryEscape(config.Password), address, config.Database)
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	return stdlib.OpenDB(*connConfig, stdlib.OptionAfterConnect(func(_ context.Context, conn *pgx.Conn) error {
		registerUint64Bitcast(conn.TypeMap())
		return nil
	})), nil
}

func registerUint64Bitcast(tm *pgtype.Map) {
	tm.RegisterType(&pgtype.Type{Name: "int8", OID: pgtype.Int8OID, Codec: uint64BitcastCodec{}})
	tm.RegisterDefaultPgType(uint64(0), "int8")
}

// uint64BitcastCodec extends the int8 codec to accept uint64 parameters via
// two's-complement bit-cast, preserving the full 64-bit range in a signed bigint.
type uint64BitcastCodec struct {
	pgtype.Int8Codec
}

func (c uint64BitcastCodec) PlanEncode(m *pgtype.Map, oid uint32, format int16, value any) pgtype.EncodePlan {
	if _, ok := value.(uint64); ok {
		if inner := c.Int8Codec.PlanEncode(m, oid, format, int64(0)); inner != nil {
			return uint64BitcastEncodePlan{inner: inner}
		}
		return nil
	}
	return c.Int8Codec.PlanEncode(m, oid, format, value)
}

type uint64BitcastEncodePlan struct {
	inner pgtype.EncodePlan
}

func (p uint64BitcastEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	if v, ok := value.(uint64); ok {
		value = int64(v)
	}
	return p.inner.Encode(value, buf)
}
