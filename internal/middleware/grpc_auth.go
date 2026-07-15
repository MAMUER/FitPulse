package middleware

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/auth"
)

const authMetadataKey = "authorization"

func GRPCAuthInterceptor(publicKeyPEM string, log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Warn("Missing gRPC metadata", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := ""
		for _, v := range md.Get(authMetadataKey) {
			if v != "" {
				authHeader = v
				break
			}
		}

		if authHeader == "" {
			log.Warn("Missing authorization header", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			log.Warn("Invalid authorization format", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}

		token := parts[1]
		claims, err := auth.ValidateAccessToken(token, publicKeyPEM)
		if err != nil {
			log.Warn("Invalid access token", zap.Error(err), zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)

		return handler(ctx, req)
	}
}
