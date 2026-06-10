package nerror

import "fmt"

// GRPCCode is a gRPC status code. The constants below mirror the canonical
// codes from google.golang.org/grpc/codes, so a GRPCCode converts directly:
//
//	codes.Code(err.GRPCCode())
//
// neji defines its own type to stay dependency-free for HTTP-only users.
type GRPCCode uint32

const (
	CodeOK                 GRPCCode = 0
	CodeCanceled           GRPCCode = 1
	CodeUnknown            GRPCCode = 2
	CodeInvalidArgument    GRPCCode = 3
	CodeDeadlineExceeded   GRPCCode = 4
	CodeNotFound           GRPCCode = 5
	CodeAlreadyExists      GRPCCode = 6
	CodePermissionDenied   GRPCCode = 7
	CodeResourceExhausted  GRPCCode = 8
	CodeFailedPrecondition GRPCCode = 9
	CodeAborted            GRPCCode = 10
	CodeOutOfRange         GRPCCode = 11
	CodeUnimplemented      GRPCCode = 12
	CodeInternal           GRPCCode = 13
	CodeUnavailable        GRPCCode = 14
	CodeDataLoss           GRPCCode = 15
	CodeUnauthenticated    GRPCCode = 16
)

var grpcCodeNames = map[GRPCCode]string{
	CodeOK:                 "OK",
	CodeCanceled:           "CANCELLED",
	CodeUnknown:            "UNKNOWN",
	CodeInvalidArgument:    "INVALID_ARGUMENT",
	CodeDeadlineExceeded:   "DEADLINE_EXCEEDED",
	CodeNotFound:           "NOT_FOUND",
	CodeAlreadyExists:      "ALREADY_EXISTS",
	CodePermissionDenied:   "PERMISSION_DENIED",
	CodeResourceExhausted:  "RESOURCE_EXHAUSTED",
	CodeFailedPrecondition: "FAILED_PRECONDITION",
	CodeAborted:            "ABORTED",
	CodeOutOfRange:         "OUT_OF_RANGE",
	CodeUnimplemented:      "UNIMPLEMENTED",
	CodeInternal:           "INTERNAL",
	CodeUnavailable:        "UNAVAILABLE",
	CodeDataLoss:           "DATA_LOSS",
	CodeUnauthenticated:    "UNAUTHENTICATED",
}

// String returns the canonical gRPC code name, e.g. "NOT_FOUND".
func (g GRPCCode) String() string {
	if name, ok := grpcCodeNames[g]; ok {
		return name
	}
	return fmt.Sprintf("CODE(%d)", uint32(g))
}

// grpcFromHTTP derives a gRPC code from an HTTP status, following the
// google.rpc HTTP mapping. Unmapped statuses fall back to
// FAILED_PRECONDITION for 4xx, INTERNAL for 5xx and UNKNOWN otherwise.
func grpcFromHTTP(status int) GRPCCode {
	switch status {
	case 400, 422:
		return CodeInvalidArgument
	case 401:
		return CodeUnauthenticated
	case 403:
		return CodePermissionDenied
	case 404:
		return CodeNotFound
	case 408, 504:
		return CodeDeadlineExceeded
	case 409:
		return CodeAborted
	case 412:
		return CodeFailedPrecondition
	case 416:
		return CodeOutOfRange
	case 429:
		return CodeResourceExhausted
	case 499:
		return CodeCanceled
	case 500:
		return CodeInternal
	case 501:
		return CodeUnimplemented
	case 503:
		return CodeUnavailable
	}
	switch {
	case status >= 400 && status < 500:
		return CodeFailedPrecondition
	case status >= 500 && status < 600:
		return CodeInternal
	}
	return CodeUnknown
}
