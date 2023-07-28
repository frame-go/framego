package errors

import (
	stderrors "errors"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/frame-go/framego/errors/proto"
)

const InvalidCode = -1

var grpcDebugMode = false

type GRPCError interface {
	GRPCStatus() *status.Status
}

type Error struct {
	error
	GRPCError

	message    string
	code       int
	cause      error
	detail     map[string]interface{}
	grpcStatus *status.Status
}

// Error implements error interface
func (e *Error) Error() string {
	if e.cause == nil {
		return e.message
	}
	return e.message + ": " + e.cause.Error()
}

// GRPCStatus implements GrpcError interface.
// Returning this error in grpc API will add detail and cause into details field in GRPC error response.
// Once GRPCStatus is called, the status data is cached, new changes will not be reflected in new calls.
func (e *Error) GRPCStatus() *status.Status {
	if e.grpcStatus != nil {
		return e.grpcStatus
	}
	if e.code == InvalidCode {
		// Try to call GRPCStatus from upper layer errors if code was not set.
		if se, ok := e.cause.(interface {
			GRPCStatus() *status.Status
		}); ok {
			e.grpcStatus = se.GRPCStatus()
			return e.grpcStatus
		}
	}
	var code codes.Code
	if e.code == InvalidCode {
		code = codes.Unknown
	} else {
		code = codes.Code(e.code)
	}
	e.grpcStatus = status.New(code, e.message)
	newStatus, err := e.grpcStatus.WithDetails(e.detailProto())
	if err == nil {
		e.grpcStatus = newStatus
	}
	return e.grpcStatus
}

func (e *Error) detailProto() *pb.Error {
	detail := &pb.Error{
		Error: e.message,
	}
	if e.detail != nil {
		detail.Detail, _ = structpb.NewStruct(e.detail)
	}
	if grpcDebugMode && e.cause != nil {
		errorCause, ok := e.cause.(*Error)
		if ok {
			detail.Cause = errorCause.detailProto()
		} else {
			detail.Cause = &pb.Error{
				Error: e.cause.Error(),
			}
		}
	}
	return detail
}

// Message returns message of error
func (e *Error) Message() string {
	return e.message
}

// Code returns code of error
func (e *Error) Code() int {
	return e.code
}

// Cause returns error cause of error
func (e *Error) Cause() error {
	return e.cause
}

// Unwrap returns error cause of error to implement Unwrap interface
func (e *Error) Unwrap() error {
	return e.cause
}

// WithCode updates code in error
func (e *Error) WithCode(code int) *Error {
	e.grpcStatus = nil //reset cache
	e.code = code
	return e
}

// WithGRPCCode updates GRPC code in error
func (e *Error) WithGRPCCode(code codes.Code) *Error {
	e.grpcStatus = nil //reset cache
	e.code = int(code)
	return e
}

// With adds custom field in detail
func (e *Error) With(key string, value interface{}) *Error {
	e.grpcStatus = nil //reset cache
	if e.detail == nil {
		e.detail = make(map[string]interface{})
	}
	e.detail[key] = value
	return e
}

// Log adds error with detail fields in Zerolog logger
func (e *Error) Log(log *zerolog.Event) *zerolog.Event {
	log.Err(e)
	e.addDetailToLogger(log)
	return log
}

func (e *Error) addDetailToLogger(log *zerolog.Event) {
	if e.cause != nil {
		errorCause, ok := e.cause.(*Error)
		if ok {
			errorCause.addDetailToLogger(log)
		}
	}
	if e.detail != nil {
		for key, value := range e.detail {
			log.Interface(key, value)
		}
	}
}

// New creates new error
func New(message string) *Error {
	return &Error{
		message: message,
		code:    InvalidCode,
	}
}

// Wrap create new error with underlying error cause
func Wrap(e error, message string) *Error {
	return &Error{
		message: message,
		code:    InvalidCode,
		cause:   e,
	}
}

// LogError adds error with detail fields in Zerolog logger
func LogError(log *zerolog.Event, err error) *zerolog.Event {
	log.Err(err)
	detailErr, ok := err.(*Error)
	if ok {
		detailErr.addDetailToLogger(log)
	}
	return log
}

// SetGRPCDebugMode sets grpc error debug mode.
// If enable, GRPCStatus will include error cause in details.
func SetGRPCDebugMode(enable bool) {
	grpcDebugMode = enable
}

// Is exports Is from std errors.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// Unwrap exports Unwrap from std errors.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// As exports As from std errors.
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}
