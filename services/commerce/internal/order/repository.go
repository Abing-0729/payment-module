package order

import (
	"context"
	"kratos-payment-lab/services/commerce/internal/platform/postgres"
	"kratos-payment-lab/services/commerce/internal/platform/postgres/query"
)

type Repository struct {
	db *postgres.Postgres
}

// queries 方法用于根据上下文返回相应的查询实例
// 它会检查上下文中是否包含事务，如果有则使用事务，否则使用数据库连接池
//
// 参数:
//
//	ctx - context.Context，上下文对象，可能包含事务信息
//
// 返回值:
//
//	*query.Queries - 返回一个查询实例，可能是基于事务的，也可能是基于数据库连接池的
func (r *Repository) queries(ctx context.Context) *query.Queries {
	// 检查上下文中是否包含 postgres 事务
	// FromContext 函数尝试从上下文中提取事务对象
	// 如果 ok 为 true，表示上下文中包含事务
	if tx, ok := postgres.FromContext(ctx); ok {
		// 如果存在事务，则使用该事务创建新的查询实例
		return query.New(tx)
	}
	// 如果不存在事务，则使用数据库连接池创建新的查询实例
	return query.New(r.db.Pool())

}
