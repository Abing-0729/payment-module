package server

import (
	"context"
	"fmt"
	"time"

	"github.com/google/wire"
	"kratos-payment-lab/services/commerce/internal/platform/postgres"
)

// ProviderSet 供 cmd 的 wire 注入使用。
// 移除其中任一 Provider(如 NewHTTPServer),make wire 将生成失败。
var ProviderSet = wire.NewSet(NewHealthService, NewHTTPServer, NewGRPCServer, NewServer, NewPostgres)

func NewServer(conf *Bootstrap) *Server { return conf.Server }

func NewPostgres(conf *Bootstrap) (*postgres.Postgres, func(), error) {
	if conf == nil || conf.Data == nil || conf.Data.Database == nil {
		return nil, func() {}, fmt.Errorf("database configuration is required")
	}
	db := conf.Data.Database
	pg, err := postgres.New(context.Background(), db.DSN, db.MaxConns, time.Duration(db.ConnectTimeout))
	if err != nil {
		return nil, func() {}, err
	}
	return pg, pg.Close, nil
}
