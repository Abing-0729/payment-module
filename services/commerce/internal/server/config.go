package server

import (
	"encoding/json"
	"time"
)

// Duration 包装 time.Duration,支持从 "10s" 字符串反序列化。
// Kratos config 内部走 JSON 解析,字符串无法直接赋给 time.Duration,
// 故实现 UnmarshalJSON 手动 ParseDuration。
type Duration time.Duration

// UnmarshalJSON 解析 "10s" 形式的时长字符串。
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	du, err := time.ParseDuration(v)
	if err != nil {
		return err
	}
	*d = Duration(du)
	return nil
}

// 配置结构体
// Bootstrap 是 configs/config.yaml 的完整映射。
// 采用纯 Go 结构体而非 conf.proto 生成代码:buf 流水线只覆盖 api/,
// 且当前没有 proto 化配置契约的需求;Issue 2 接入数据库时扩展 Data 段。
type Bootstrap struct {
	Server *Server `yaml:"server"`
	Data   *Data   `yaml:"data"`
}

type Data struct {
	Database *Database `yaml:"database"`
}

type Database struct {
	DSN            string   `yaml:"dsn"`
	MaxConns       int32    `yaml:"max_conns"`
	ConnectTimeout Duration `yaml:"connect_timeout"`
}

// Server 承载 HTTP 与 gRPC 两套监听配置。
type Server struct {
	HTTP *HTTP `yaml:"http"`
	GRPC *GRPC `yaml:"grpc"`
}

// HTTP 对应 config.yaml 的 server.http。
type HTTP struct {
	Network string   `yaml:"network"`
	Addr    string   `yaml:"addr"`
	Timeout Duration `yaml:"timeout"`
}

// GRPC 对应 config.yaml 的 server.grpc。
type GRPC struct {
	Network string   `yaml:"network"`
	Addr    string   `yaml:"addr"`
	Timeout Duration `yaml:"timeout"`
}
