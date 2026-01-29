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

// toJSON converts a value to its JSON representation for logging.
func toJSON(v any) string {
	if v == nil {
		return "null"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<error: %v>", err)
	}
	return string(data)
}

// loggingInterceptor logs gRPC requests and responses using JSON format
func loggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			// Log request as JSON
			logger.Info("RPC request: %s req=%s", procedure, toJSON(req.Any()))

			// Call the handler
			resp, err := next(ctx, req)

			// Log response as JSON
			duration := time.Since(start)
			if err != nil {
				logger.Info("RPC response: %s err=%q duration=%v", procedure, err.Error(), duration)
			} else {
				logger.Info("RPC response: %s resp=%s duration=%v", procedure, toJSON(resp.Any()), duration)
			}

			return resp, err
		}
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
