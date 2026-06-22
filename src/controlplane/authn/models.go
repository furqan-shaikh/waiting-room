package authn

import (
	"context"
	"errors"
)

type contextKey string

const userPrincipalKey contextKey = "user"
const MaxAllowedClockSkew = 1

type UserPrincipal struct {
	Id                   string
	Name                 string
	AuthenticationMethod string
}

func GetUserPrincipal(ctx context.Context) (UserPrincipal, error) {
	ctxUserVal := ctx.Value(userPrincipalKey)
	userPrincipal, ok := ctxUserVal.(UserPrincipal)
	if !ok {
		return UserPrincipal{}, errors.New("Unauthorized: User Principal missing")
	}
	return userPrincipal, nil
}

func SetUserPrincipal(ctx context.Context, userPrincipal UserPrincipal) context.Context {
	return context.WithValue(ctx, userPrincipalKey, userPrincipal)
}
