package server

import (
	"context"

	v1 "kratos-payment-lab/api/commerce/v1"
)

// serviceName 写入健康检查响应,便于区分两个进程。
const serviceName = "commerce"

// HealthService 实现 api/v1 的 HealthService(HTTP /healthz 与 gRPC)。
// Issue 1 为存活级:进程存活即返回 SERVING;
// Issue 2 接入 PostgreSQL 后升级为就绪检查(Check 内部 ping DB/Redis)。
type HealthService struct {
	v1.UnimplementedHealthServiceServer
}

// NewHealthService 构造健康检查服务。
func NewHealthService() *HealthService {
	return &HealthService{}
}

// Check 返回服务名与存活状态。
func (s *HealthService) Check(_ context.Context, _ *v1.CheckRequest) (*v1.CheckResponse, error) {
	return &v1.CheckResponse{
		Service: serviceName,
		Status:  "SERVING",
	}, nil
}
