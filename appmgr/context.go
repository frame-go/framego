package appmgr

import (
	"context"

	"github.com/gin-gonic/gin"
)

type IContextGetter interface {
	Value(key interface{}) interface{}
}

const contextServiceKey = "_service"

func GetContextService(ctx IContextGetter) Service {
	v := ctx.Value(contextServiceKey)
	if v == nil {
		return nil
	}
	ctxService, ok := v.(Service)
	if !ok || ctxService == nil {
		return nil
	}
	return ctxService
}

func SetContextService(ctx context.Context, service Service) context.Context {
	return context.WithValue(ctx, contextServiceKey, service)
}

func SetGinContextService(ctx *gin.Context, service Service) {
	ctx.Set(contextServiceKey, service)
}
