package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/src/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// loggingInterceptor logs gRPC requests and responses using JSON format
func loggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()

			// Call the handler
			response, err := next(ctx, request)

			duration := time.Since(start)

			if err != nil {
				logGrpcError(duration, request, err)
			} else {
				logGrpcResponse(duration, request, response)
			}

			return response, err
		}
	}
}

func logGrpcError(duration time.Duration, req connect.AnyRequest, err error) bool {
	// marshall request into a single line
	reqMsg, ok := req.Any().(proto.Message)
	reqMsgJSON, _ := protojson.MarshalOptions{}.Marshal(reqMsg)
	preconditions.Check(ok, "This must always be a proto.Message")

	return logger.Info(
		"GRPC %s duration=%s req=%s err=%q",
		req.Spec().Procedure,
		formatDuration(duration),
		reqMsgJSON,
		err.Error(),
	)
}

func logGrpcResponse(duration time.Duration, req connect.AnyRequest, resp connect.AnyResponse) {
	// marshall request into a single line
	reqMsg, ok := req.Any().(proto.Message)
	reqMsgJSON, _ := protojson.MarshalOptions{}.Marshal(reqMsg)
	preconditions.Check(ok, "This must always be a proto.Message")

	respMsg, ok := resp.Any().(proto.Message)
	preconditions.Check(ok, "This must always be a proto.Message")

	// Log response, but include the result only in DEBUG level.
	if logger.Level() > logger.DEBUG {
		logger.Info(
			"GRPC %s duration=%s (%d bytes) %s",
			req.Spec().Procedure,
			formatDuration(duration),
			proto.Size(respMsg),
			reqMsgJSON,
		)
	} else {
		logger.Debug(
			"GRPC %s duration=%s (%d bytes) %s resp=%s",
			req.Spec().Procedure,
			formatDuration(duration),
			proto.Size(respMsg),
			reqMsgJSON,
			respMsg,
		)
	}
}

func formatDuration(duration time.Duration) string {
	if duration >= time.Millisecond {
		return fmt.Sprintf("%.3fms", float64(duration/time.Millisecond))
	} else {
		return fmt.Sprintf("%dµs", int32(duration/time.Microsecond))
	}
}

// validatorInterceptor validates requests against JSON schemas
func validatorInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			if validationErr := validateRequest(procedure, req.Any()); validationErr != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, validationErr)
			}

			return next(ctx, req)
		}
	}
}

// protoToMap converts a protobuf message to a map for validation.
func protoToMap(msg any) (map[string]any, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// validateRequest validates a request against its JSON schema.
// Returns an error suitable for connect.NewError if validation fails.
func validateRequest(procedure string, msg any) error {
	reqMap, err := protoToMap(msg)
	if err != nil {
		return fmt.Errorf("failed to parse request: %w", err)
	}

	if err := api.ValidateRequest(procedure, reqMap); err != nil {
		return fmt.Errorf("%s", api.FormatValidationError(err))
	}
	return nil
}
