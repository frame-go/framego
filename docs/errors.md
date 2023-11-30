# Framego Errors Module

## Introduction

`framego/errors` module provides useful utilities for error handling and is compatible with errors module in standard library.

## Package

### Functions

```go
// New creates new error
func New(message string) *Error
 
// Wrap create new error with underlying error cause
func Wrap(e error, message string) *Error

// LogError adds error with detail fields in Zerolog logger
func LogError(log *zerolog.Event, err error) *zerolog.Event

// Is exports Is from std errors.
func Is(err, target error) bool

// Unwrap exports Unwrap from std errors.
func Unwrap(err error) error

// As exports As from std errors.
func As(err error, target interface{}) bool
```

## Error Structure

```go
// Error implements error interface
func (e *Error) Error() string

// GRPCStatus implements GrpcError interface.
// Returning this error in grpc API will add detail and cause into details field in GRPC error response.
// Once GRPCStatus is called, the status data is cached, new changes will not be reflected in new calls.
func (e *Error) GRPCStatus() *status.Status

// Message returns message of error
func (e *Error) Message() string

// Code returns code of error
func (e *Error) Code() int

// Cause returns error cause of error
func (e *Error) Cause() error

// Unwrap returns error cause of error to implement Unwrap interface
func (e *Error) Unwrap() error

// WithCode updates code in error
func (e *Error) WithCode(code int) *Error

// WithGRPCCode updates GRPC code in error
func (e *Error) WithGRPCCode(code codes.Code) *Error 

// With adds custom field in detail
func (e *Error) With(key string, value interface{}) *Error

// Log adds error with detail fields in Zerolog logger
func (e *Error) Log(log *zerolog.Event) *zerolog.Event
```

## Features

### Wrap Error

Error can be created by `New` method or wrapped from an underlying error by `Wrap` method. `message` is the keyword of error for the client to recognize the error. It is suggested to be `snake_case`. 

It is suggested to only write error logs in the high layer function. Low layer function can wrap error got from underlying functions add related fields and return to caller function. Thus, the full error information will be kept.

**New error**

```go
return nil errors.New("object_not_found").With("name" name)
```

**Wrap error**

```go
obj, err := db.GetObjectByName(name)
if err != nil {
    return nil errors.Wrap(err, "object_not_found").With("name" name)
}
```

### With Fields

An error structure can attach many detail fields related to the error.
- `With` method can be called many times to add detail fields into error structure.
- Error fields will be added to log messages created by `errors.LogError` method.
- Error fields will be added to `detail` field in GRPC response.

### Logging Integration

Use `errors.LogError` method to write error details in logger of `zerolog`.
- Wrapped errors will be appended to the error message.
- Error fields will be added as log message fields.

**Code**

```go
resp, err := service.Query(req)
if err != nil {
    errors.LogError(log.Error(), err).Msg("query_error")
    return err
}
```

**Log**

```
2023-01-1T12:00:00+08:00 ERR query_error="get_object_error: query_db_error: Error 2013: Lost connection to MySQL server during query" ip=127.0.0.1 method=/sample.Sample/GetObject name=test_object request_id=5a383bdf-fd78-4122-8f33-8bd25d78baac
```

### GRPC Support

Call `WithGRPCCode` to set a GRPC error code in the error structure.
- `code` is the GRPC code for the error.
- If not set, the default code is `codes.Unknown`.
- The error with GRPC code can be handled by the GRPC library.

**Code**

```go
obj, err := service.GetObjectByName(name)
if err != nil {
    return nil errors.Wrap(err, "object_not_found").WithGRPCCode(codes.NotFound).With("name" name)
}
```

**Error Response Structure**

- `message`: top-level error message
- `cause`: wrapped error
- `detail`: with fields

**Error Response Example**

```json
{
    "code": 5,
    "message": "object_not_found",
    "details": [
        {
            "@type": "type.googleapis.com/errors.Error",
            "error": "object_not_found",
            "detail": {
                "name": "test"
            },
            "cause": {
                "error": "record not found",
                "detail": null,
                "cause": null
            }
        }
    ]
}
```
