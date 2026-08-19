package main

import (
	"flag"
	"kratos-payment-lab/services/mockpay/internal/server"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// go build -ldflags "-X main.Version=x.y.z" -o mockpay
var (
	Name     = "mockpay"
	Version  = "v1.0.0"
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")

}

func main() {
	flag.Parse()

	//1.日志器
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp, "caller", log.DefaultCaller)

	//2.配置源
	c := config.New(config.WithSource(file.NewSource(flagconf)))
	defer c.Close()
	if err := c.Load(); err != nil {
		panic(err)
	}

	//3.yaml反序列化到server.Bootstrap 结构体
	var bc server.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	//4.交给wire:wireApp在wire_gen.go里由wire生成
	// 返回kratos.APP+清理函数
	app, cleanup, err := wireApp(bc.Server, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func newApp(logger log.Logger, hs *http.Server, gs *grpc.Server) *kratos.App {
	return kratos.New(
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			gs,
		),
	)
}
