package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres 封装 pgx 连接池。
type Postgres struct {
	pool *pgxpool.Pool
}

// New 创建连接池并预检连通性。
// maxConns: 池内最大连接数（防止连接风暴打垮数据库）；
// connectTimeout: 单次建连超时（数据库挂了要快速失败，而不是无限等）。
func New(ctx context.Context, dsn string, maxConns int32, connectTimeout time.Duration) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析 DSN: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.ConnConfig.ConnectTimeout = connectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建连接池: %w", err)
	}
	// 预检：启动即验证数据库可达，而不是第一个请求才暴露问题
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping 数据库: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// Close 释放连接池。服务优雅退出时调用。
func (p *Postgres) Close() { p.pool.Close() }

// Pool 暴露底层池。sqlc 生成的 DAO 绑定它执行普通查询。
func (p *Postgres) Pool() *pgxpool.Pool { return p.pool }
