package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
)

// loggingInterceptor logs gRPC requests and responses using JSON format
func loggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			// Log request as JSON
			if !logger.Debug("RPC request: %s req=%s", procedure, req.Any()){
				logger.Info("RPC request: %s", procedure)
			}

			// Call the handler
			resp, err := next(ctx, req)

			// Log response as JSON
			duration := formatDuration(start)
			if err != nil {
				logger.Info("RPC response: %s duration=%s err=%q", procedure, duration, err.Error())
			} else {
				if !logger.Debug("RPC response: %s duration=%s resp=%s", procedure, duration, resp.Any()) {
					logger.Info("RPC response: %s duration=%s", procedure, duration)
				}
			}

			return resp, err
		}
	}
}

func formatDuration(start time.Time) string {
	duration := time.Since(start)
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
