package authz

import (
	"context"

	libctx "dislyze/jirachi/ctx"
)

func GetIPWhitelistActive(ctx context.Context) bool {
	ipConfig := libctx.GetIPWhitelistConfig(ctx)
	return ipConfig.Active
}
