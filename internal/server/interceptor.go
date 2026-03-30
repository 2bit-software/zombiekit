package server

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/2bit-software/zombiekit/internal/logging"
)

func NewLoggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			resp, err := next(ctx, req)

			duration := time.Since(start)
			attrs := []any{
				slog.String("procedure", procedure),
				slog.Duration("duration", duration),
			}

			if err != nil {
				code := connect.CodeOf(err)
				attrs = append(attrs,
					slog.String("code", code.String()),
					slog.String("error", err.Error()),
				)
				logging.Logger().Warn("rpc failed", attrs...)
			} else {
				attrs = append(attrs, slog.String("code", "ok"))
				logging.Logger().Info("rpc", attrs...)
			}

			return resp, err
		}
	}
}
