package auth

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Authenticator struct {
	Token string
}

func (a Authenticator) Authenticate(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "no headers in request")
	}

	authHeaders, ok := md["nekoray_auth"]
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "no header in request")
	}

	// 严格保持原有的单值检查逻辑
	if len(authHeaders) != 1 {
		return ctx, status.Error(codes.Unauthenticated, "more than 1 header in request")
	}

	if authHeaders[0] != a.Token {
		return ctx, status.Error(codes.Unauthenticated, "invalid token")
	}

	// 严格保持原有的 Header 清除逻辑，确保后续 context 安全
	mdCopy := md.Copy()
	mdCopy["nekoray_auth"] = nil
	newCtx := metadata.NewIncomingContext(ctx, mdCopy)

	return newCtx, nil
}
