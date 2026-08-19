package server

import "github.com/google/wire"

// ProviderSet 供 cmd 的 wire 注入使用。
// 移除其中任一 Provider(如 NewHTTPServer),make wire 将生成失败。
var ProviderSet = wire.NewSet(NewHealthService, NewHTTPServer, NewGRPCServer)
