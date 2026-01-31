package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxLogLength = 200

// toJSON converts a value to its JSON representation for logging.
// Uses protojson for protobuf messages (canonical conversion).
// Truncates output to maxLogLength characters.
func toJSON(v any) string {
	if v == nil {
		return "null"
	}

	var s string
	if msg, ok := v.(proto.Message); ok {
		// Use canonical protojson for protobuf messages
		s = protojson.Format(msg)
	} else {
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("<error: %v>", err)
		}
		s = string(data)
	}

	return truncate(s, maxLogLength)
}

// truncate cuts the string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
			duration := formatDuration(start)
			if err != nil {
				logger.Info("RPC response: %s err=%q duration=%s", procedure, err.Error(), duration)
			} else {
				logger.Info("RPC response: %s resp=%s duration=%s", procedure, toJSON(resp.Any()), duration)
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
