package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Transactor 定义事务边界抽象：业务层只依赖这个接口，不依赖具体实现。
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type txKey struct{}

// WithinTransaction 在事务中执行 fn。
//
// 关键设计：*pgx.Tx 通过 context.Context 传递。
// 业务代码无论调用链多深，只要从 ctx 取事务绑定对象，
// 整条链就复用同一个连接（同一个事务）。
// 这保证：事务内两次查询看到彼此未提交的修改——这正是"事务"的意义。
func (p *Postgres) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	// 已在事务中则直接复用（支持嵌套调用，如 service 内再调一个事务方法）
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务: %w", err)
	}
	// defer 回滚：fn 成功且 Commit 成功后，这里的 Rollback 返回 ErrTxClosed，无害；
	// fn 返回错误时，这里就是真正的回滚。
	defer tx.Rollback(context.WithoutCancel(ctx))

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err // 不包装，保留原始错误给上层
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}

// FromContext 从上下文取出事务。sqlc 生成的 DAO 用它在事务内执行查询。
func FromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
