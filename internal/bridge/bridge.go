package bridge

import (
	"context"

	"github.com/autopeer-io/autopeer/internal/bridge/k8s"
	"github.com/autopeer-io/autopeer/internal/bridge/server"
	"github.com/autopeer-io/autopeer/pkg/log"
)

// CloudHubServer is the main application struct for CloudHub.
type CloudHubServer struct {
	serverManager *server.Manager
	k8sPipeline   *k8s.StatusPipeline
}

// Run starts the application components.
func (a *CloudHubServer) Run(ctx context.Context) error {
	log.Info("Starting CloudHub Application...")
	// 1. 启动 Pipeline (后台)
	go a.k8sPipeline.Start(ctx)

	// 2. 启动 Servers (阻塞)
	err := a.serverManager.Start(ctx)

	// 3. Server 退出后，给 Pipeline 一点时间刷写数据 (可选)
	// 由于 Pipeline 监听了 ctx.Done，它会尝试最后一次 flush。
	// 但如果 serverManager 退出是因为 error 而不是 ctx cancel，
	// 这里可能需要手动 cancel 一个专门控制 Pipeline 的 context。

	return err
}
