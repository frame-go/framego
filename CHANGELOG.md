# Changelog

## 0.4.0 - 2026-09-25

- Require Go 1.27 and upgrade dependencies
- Add SSE for gateway server-streaming and bidi responses when the request accepts `text/event-stream`, with a `: ping` heartbeat every 15s
- Remove the mssqlx database client (`GetDatabaseSqlxClient`, `GetSqlxClient`)
- Fix `StringMap.GetStructWithValidation` validating the global config instead of the receiver's map

## 0.3.5 - 2026-07-04

- Add PostgreSQL support to the database client (`type: postgres`), storing `uniqueid.ID` as signed bigint

## 0.3.4 - 2026-06-18

- Add `uniqueid.ID.StringOrEmpty` and `uniqueid.ParseIDOptional`

## 0.3.3 - 2025-01-08

- Upgrade gorm, the GORM MySQL driver and dbresolver

## 0.3.2 - 2025-01-08

- Downgrade grpcui from 1.5.0 to 1.4.2

## 0.3.1 - 2025-01-08

- Remove the gRPC window and buffer sizes set in 0.2.3

## 0.3.0 - 2024-07-10

- Add the `crypto` and `uniqueid` packages and the app ID generator (`id_generator` config)
- Remove the `rand` seeding at init

## 0.2.3 - 2024-07-09

- Set 1 MB window and buffer sizes for the gRPC client and server

## 0.2.2 - 2024-06-04

- Upgrade athenz to fix the Go JOSE vulnerability

## 0.2.1 - 2024-05-23

- Set the gRPC client and server max message size to 10 MB

## 0.2.0 - 2024-04-12

- Require Go 1.21 and upgrade dependencies
- Add the README and errors documentation

## 0.1.0 - 2023-07-28

- Initial release
