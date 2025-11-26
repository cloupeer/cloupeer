package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// 定义指标变量
var (
	// HubConnectivityStatus 记录 Controller 到 Hub 的连接状态
	// 1 = Ready, 0 = Not Ready (Idle, Connecting, TransientFailure)
	HubConnectivityStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cloupeer_hub_connectivity_status",
			Help: "The connectivity status to Cloupeer Hub (1=Ready, 0=NotReady).",
		},
	)

	// CommandSentTotal 记录发送命令的总数
	CommandSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloupeer_command_sent_total",
			Help: "Total number of VehicleCommands sent to Hub.",
		},
		[]string{"status", "type"}, // status: success/failed, type: OTA/Reboot
	)

	// CommandLatency 记录 gRPC 调用耗时
	CommandLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cloupeer_command_latency_seconds",
			Help:    "Latency of sending commands to Hub via gRPC.",
			Buckets: prometheus.DefBuckets, // 使用默认的桶分布 (.005, .01, .025, ...)
		},
		[]string{"type"}, // type: OTA/Reboot
	)
)

// init 函数会自动将这些指标注册到 controller-runtime 的全局 Registry 中
// 这样它们就会出现在 :8443/metrics 端点上
func init() {
	metrics.Registry.MustRegister(HubConnectivityStatus)
	metrics.Registry.MustRegister(CommandSentTotal)
	metrics.Registry.MustRegister(CommandLatency)
}
