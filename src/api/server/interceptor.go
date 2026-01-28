package server

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
)

// loggingInterceptor logs gRPC requests and responses
func loggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			// Log request
			logger.Info("RPC request: %s req=%v", procedure, req.Any())

			// Call the handler
			resp, err := next(ctx, req)

			// Log response
			duration := time.Since(start)
			if err != nil {
				logger.Info("RPC response: %s err=%v duration=%v", procedure, err, duration)
			} else {
				logger.Info("RPC response: %s resp=%v duration=%v", procedure, resp.Any(), duration)
			}

			return resp, err
		}
	}
}
