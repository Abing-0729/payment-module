package order

import (
	"context"
	stderrors "errors"

	apperrors "kratos-payment-lab/services/commerce/internal/platform/errors"
	"kratos-payment-lab/services/commerce/internal/platform/postgres"
	"kratos-payment-lab/services/commerce/internal/platform/postgres/query"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type OrderRepository interface {
	Create(context.Context, *Order) error
	GetByOrderNo(context.Context, string) (*Order, error)
	Transition(context.Context, string, []OrderStatus, OrderStatus) (bool, error)
}

// 调用 sqlc；
// 处理数据库错误；
// 在普通连接池和事务连接之间切换；
// 把数据库模型转换成领域模型。
type Repository struct {
	db *postgres.Postgres
}

func NewRepository(db *postgres.Postgres) *Repository { return &Repository{db: db} }

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

func (r *Repository) Create(ctx context.Context, order *Order) error {
	if order == nil {
		return apperrors.New(apperrors.INVALID_ARGUMENT, "order is nil")
	}
	row, err := r.queries(ctx).CreateOrder(ctx, query.CreateOrderParams{
		OrderNo: order.OrderNo, UserID: order.UserID, AmountCents: order.AmountCents, Status: order.Status.String(),
	})
	if err != nil {
		return mapDatabaseError(err)
	}
	*order = fromQuery(row)
	return nil
}

func (r *Repository) GetByOrderNo(ctx context.Context, orderNo string) (*Order, error) {
	row, err := r.queries(ctx).GetOrderByOrderNo(ctx, orderNo)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.ORDER_NOT_FOUND, "order not found")
		}
		return nil, err
	}
	result := fromQuery(row)
	return &result, nil
}

func (r *Repository) Transition(ctx context.Context, orderNo string, from []OrderStatus, to OrderStatus) (bool, error) {
	fromValues := make([]string, 0, len(from))
	for _, status := range from {
		fromValues = append(fromValues, status.String())
	}
	rows, err := r.queries(ctx).TransitionOrder(ctx, query.TransitionOrderParams{
		OrderNo: orderNo, FromStatus: fromValues, ToStatus: to.String(),
	})
	return rows == 1, err
}

func fromQuery(row query.Order) Order {
	return Order{ID: row.ID, OrderNo: row.OrderNo, UserID: row.UserID, AmountCents: row.AmountCents,
		Status: mustParseStatus(row.Status), Version: row.Version, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}

func mustParseStatus(value string) OrderStatus {
	status, _ := ParseOrderStatus(value)
	return status
}

func mapDatabaseError(err error) error {
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.New(apperrors.ALREADY_EXISTS, "order_no already exists")
	}
	return err
}
