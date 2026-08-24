//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // goose 通过 database/sql 连接
	"github.com/pressly/goose/v3"

	"kratos-payment-lab/db/migrations"
	"kratos-payment-lab/services/commerce/internal/platform/postgres/query"
)

// 测试依赖本地数据库：docker compose -f deploy/docker-compose/docker-compose.yml up -d postgres
var (
	dsn string
	p   *Postgres
)

func TestMain(m *testing.M) {
	dsn = os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://payment:payment@localhost:5432/payment?sslmode=disable"
	}

	// 场景一/二的入口：空数据库完整迁移到最新版本
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}
	goose.SetBaseFS(migrations.FS)
	goose.SetDialect("postgres")
	if err := goose.Up(db, "."); err != nil {
		fmt.Printf("goose up 失败: %v\n", err)
		os.Exit(1)
	}
	db.Close()

	// 用生产代码路径创建连接池（覆盖连接池参数配置）
	p, err = New(context.Background(), dsn, 10, 5*time.Second)
	if err != nil {
		fmt.Printf("创建连接池失败: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	p.Close()
	os.Exit(code)
}

// TestMigrationsUpDown：验证 down 到底后表消失、重新 up 后恢复。
// 即"空数据库可完整迁移"+"down 后可重新 up"两个验收场景。
func TestMigrationsUpDown(t *testing.T) {
	tableExists := func(db *sql.DB) bool {
		var exists bool
		err := db.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'accounts')`,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("查询表是否存在失败: %v", err)
		}
		return exists
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// TestMain 已 up：表应在
	if !tableExists(db) {
		t.Fatal("迁移后 accounts 表不存在")
	}

	// down 到底 → 表消失
	if err := goose.DownTo(db, ".", 0); err != nil {
		t.Fatalf("goose down 失败: %v", err)
	}
	if tableExists(db) {
		t.Fatal("goose down 后 accounts 表仍存在")
	}

	// 重新 up → 表恢复（等价于空库完整迁移）
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up 失败: %v", err)
	}
	if !tableExists(db) {
		t.Fatal("重新 up 后 accounts 表不存在")
	}
}

// TestRollbackOnError：事务回调返回错误时，事务内全部修改回滚。
func TestRollbackOnError(t *testing.T) {
	err := p.WithinTransaction(context.Background(), func(ctx context.Context) error {
		tx, ok := FromContext(ctx)
		if !ok {
			t.Fatal("事务上下文中取不到 tx")
		}

		// sqlc 生成的 DAO 可以绑定到事务连接（pgx.Tx 满足 DBTX 接口）
		if v, err := query.New(tx).Ping(ctx); err != nil || v != 1 {
			return fmt.Errorf("事务内 Ping 失败: v=%d err=%v", v, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO accounts (balance) VALUES (100)`); err != nil {
			return err
		}
		return errors.New("业务逻辑失败")
	})
	if err == nil {
		t.Fatal("期望事务返回错误，实际为 nil")
	}

	// 事务已回滚：accounts 表应为空
	var n int
	if err := p.Pool().QueryRow(context.Background(), `SELECT COUNT(*) FROM accounts`).Scan(&n); err != nil {
		t.Fatalf("查询行数失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("期望回滚后 0 行，实际 %d 行", n)
	}
}

// TestTransactionContextCancel：事务中 context 被取消时，事务终止且数据回滚。
func TestTransactionContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	err := p.WithinTransaction(ctx, func(ctx context.Context) error {
		tx, ok := FromContext(ctx)
		if !ok {
			t.Fatal("事务上下文中取不到 tx")
		}

		// 第一笔插入成功（尚未提交）
		if _, err := tx.Exec(ctx, `INSERT INTO accounts (balance) VALUES (1)`); err != nil {
			return err
		}

		// 事务中途取消上下文 → 后续 SQL 立即失败
		cancel()
		if _, err := tx.Exec(ctx, `INSERT INTO accounts (balance) VALUES (2)`); err == nil {
			return errors.New("context 已取消但 Exec 未失败")
		}
		return errors.New("模拟：把取消后的错误传回去")
	})
	if err == nil {
		t.Fatal("期望事务返回错误，实际为 nil")
	}

	var n int
	if err := p.Pool().QueryRow(context.Background(), `SELECT COUNT(*) FROM accounts`).Scan(&n); err != nil {
		t.Fatalf("查询行数失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("期望取消后回滚 0 行，实际 %d 行", n)
	}
}

// TestPing：sqlc 生成的查询代码在连接池上可用。
func TestPing(t *testing.T) {
	v, err := query.New(p.Pool()).Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping 失败: %v", err)
	}
	if v != 1 {
		t.Fatalf("Ping 期望 1，实际 %v", v)
	}
}
