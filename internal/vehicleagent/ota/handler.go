package ota

import (
	"context"
	"fmt"
	"time"

	pb "github.com/autopeer-io/autopeer/api/proto/v1"
	"github.com/autopeer-io/autopeer/pkg/log"
)

func (m *Manager) HandleCommand(ctx context.Context, cmd *pb.AgentCommand) error {
	log.Info(">>> PROCESSING COMMAND <<<",
		"Type", cmd.CommandType,
		"ID", cmd.CommandName,
		"Params", cmd.Parameters,
		"Time", time.Unix(cmd.Timestamp, 0).Format(time.RFC3339))

	if cmd.CommandType != "OTA" {
		return nil
	}

	// 这里是根据架构设计的后续步骤：
	// 1. "触发一条消息提醒车主" -> Log / UI Event
	// 2. "车主点击升级" -> 模拟等待或直接调用
	go m.execute(ctx, cmd)

	return nil
}

func (m *Manager) HandleResponse(ctx context.Context, resp *pb.OTAResponse) error {
	fmt.Printf("Got URL: %s\n", resp.DownloadUrl)
	m.lock.Lock()
	if ch, ok := m.pending[resp.RequestId]; ok {
		ch <- resp.DownloadUrl
		delete(m.pending, resp.RequestId) // 清理
	}
	m.lock.Unlock()
	return nil
}
