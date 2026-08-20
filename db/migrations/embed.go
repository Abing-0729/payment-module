package migrations

import "embed"

// FS 嵌入迁移文件，供测试通过 goose 库 API 执行迁移。
// 注意：go:embed 不能跨目录，所以这个文件必须与迁移 SQL 同目录。
//
//go:embed *.sql
var FS embed.FS
