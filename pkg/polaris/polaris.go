package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"resty.dev/v3"

	polaris "github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

// PolarisLB 实现 resty.LoadBalancer，使用 Polaris 做服务发现和实例选择。
type PolarisLB struct {
	mu sync.Mutex

	namespace string
	service   string
	protocol  string // "http" or "https"

	consumer polaris.ConsumerAPI

	// 可选：缓存最近一次选择的实例，用于 Feedback 上报
	lastInst *model.Instance
}

// 确保实现接口
var _ resty.LoadBalancer = (*PolarisLB)(nil)

// NewPolarisLB 创建 Polaris 负载均衡器
func NewPolarisLB(namespace, service, protocol string, addr string) (*PolarisLB, error) {
	consumer, err := polaris.NewConsumerAPIByAddress(addr)
	if err != nil {
		return nil, err
	}

	return &PolarisLB{
		namespace: namespace,
		service:   service,
		protocol:  protocol,
		consumer:  consumer,
	}, nil
}

// Next 从 Polaris 选择一个实例，并返回 BaseURL
func (m *PolarisLB) Next() (string, error) {
	// 按需设置请求（可添加路由元数据、调用标签等）
	req := &polaris.GetOneInstanceRequest{}
	req.Service = m.service
	req.Namespace = m.namespace

	resp, err := m.consumer.GetOneInstance(req)
	if err != nil {
		return "", fmt.Errorf("polaris get instance failed: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("polaris returned no instance")
	}

	inst := resp.GetInstance()
	baseURL := fmt.Sprintf("%s://%s:%d", m.protocol, inst.GetHost(), inst.GetPort())

	// 保存实例用于后续 Feedback 上报
	m.mu.Lock()
	m.lastInst = &inst
	m.mu.Unlock()

	return baseURL, nil
}

// Feedback 接收 Resty 的请求结果反馈，可将调用结果上报给 Polaris
func (m *PolarisLB) Feedback(rf *resty.RequestFeedback) {
	// 可根据 rf 信息判断成功/失败，并上报给 Polaris
	m.mu.Lock()
	inst := m.lastInst
	m.mu.Unlock()
	if inst == nil {
		return
	}

	status := api.RetSuccess
	if !rf.Success {
		status = api.RetFail
	}

	var a int32 = 10
	c := 10 * time.Second
	result := &polaris.ServiceCallResult{
		ServiceCallResult: model.ServiceCallResult{
			EmptyInstanceGauge: model.EmptyInstanceGauge{},
			CalledInstance:     *m.lastInst,
			RetStatus:          status,
			RetCode:            &a,
			Delay:              &c,
			SourceService: &model.ServiceInfo{
				Service:   m.service,
				Namespace: m.namespace,
			},
		},
	}
	// 上报结果（忽略上报错误以保证不影响主流程）
	err := m.consumer.UpdateServiceCallResult(result)
	fmt.Println("11111111111111111111111111111111111111", err)
}

// Close 释放 Polaris SDK 资源
func (m *PolarisLB) Close() error {
	if m.consumer != nil {
		m.consumer.Destroy()
	}
	return nil
}

func main() {
	// 初始化自定义 Polaris 负载均衡器
	lb, err := NewPolarisLB(
		"default",            // Namespace
		"DiscoverEchoServer", // ServiceName
		"http",               // Protocol
		"127.0.0.1:8091",     // Polaris 控制面地址
	)
	if err != nil {
		log.Fatalf("init polaris lb failed: %v", err)
	}
	defer lb.Close()

	// 创建 Resty 客户端并注入负载均衡器
	client := resty.New().
		SetLoadBalancer(lb).
		SetTimeout(5 * time.Second).
		// 你可以设置相对路径请求，BaseURL 将由负载均衡器决定
		SetBaseURL("/") // 可设为占位，实际会被 LoadBalancer.Next() 返回值使用
	defer client.Close()

	// 示例请求：相对路径将拼接到 Polaris 选出的 BaseURL 之后
	resp, err := client.R().
		SetHeader("Accept", "application/json").
		// 注意：此处的 URL 为相对路径
		Get("/echo")
	if err != nil {
		log.Fatalf("request error: %v", err)
	}
	log.Printf("status=%d body=%s", resp.StatusCode(), resp.String())
}
