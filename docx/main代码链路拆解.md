1. 加载配置：config.New + file.NewSource 读取配置文件，c.Scan(&bc) 反序列化到 server.Bootstrap，拿到服务器相关配置（比如 HTTP 端口、gRPC 端口、数据库地址等）。

2. 依赖注入（wire）：wireApp(bc.Server, logger) 是由 wire 生成的代码（通常在 wire_gen.go）。它负责：

3. 创建数据库连接、业务 service

4. 创建 gRPC Server，并把 service 注册上去（v1.RegisterXxxServer）

5. 创建 HTTP Server，并把 service 注册上去（v1.RegisterXxxHTTPServer）

6. 可能还会添加中间件（拦截器）

7. 返回 *kratos.App 和清理函数

8. 组装 App：newApp 把 wire 传来的 hs 和 gs 以及 logger、Name、Version 统一打包成一个 kratos.App。

9. 运行：app.Run() 启动所有 server，并处理信号（优雅退出）。
