package middleware

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/auth/claims"
	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/sanitize"
)

const authMetadataKey = "authorization"

// GRPCAuthInterceptor rejects unary calls without a valid bearer token.
// The health service stays open so Kubernetes probes can reach it.
func GRPCAuthInterceptor(publicKeyPEM string, log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		claims, err := authenticateGRPC(ctx, publicKeyPEM, log, info.FullMethod)
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)

		return handler(ctx, req)
	}
}

// authenticateGRPC extracts the bearer token from the incoming metadata and
// validates it, returning the claims to attach to the request context.
func authenticateGRPC(
	ctx context.Context,
	publicKeyPEM string,
	log *zap.Logger,
	fullMethod string,
) (*claims.Claims, error) {
	method := sanitize.LogString(fullMethod)

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Warn("Missing gRPC metadata", zap.String("method", method))
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeader := firstNonEmpty(md.Get(authMetadataKey))
	if authHeader == "" {
		log.Warn("Missing authorization header", zap.String("method", method))
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		log.Warn("Invalid authorization format", zap.String("method", method))
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	tokenClaims, err := jwt.ValidateAccessToken(parts[1], publicKeyPEM)
	if err != nil {
		log.Warn("Invalid access token", zap.Error(err), zap.String("method", method))
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return tokenClaims, nil
}

// firstNonEmpty returns the first non-empty value, or "" when none is set.
func firstNonEmpty(values []string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
