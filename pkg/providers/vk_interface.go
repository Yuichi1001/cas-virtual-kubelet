package providers

import (
	"context"
	"fmt"
	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/practice/virtual-kubelet-practice/pkg/util"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
)

// CreatePod 创建pod
func (c *CasProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

// UpdatePod 更新pod
func (c *CasProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

// DeletePod 删除pod
func (c *CasProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

// GetPod 获取pod
func (c *CasProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return &corev1.Pod{}, nil
}

// GetPodStatus 获取pod状态
func (c *CasProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	return &corev1.PodStatus{}, nil
}

// GetPods 获取pod列表
func (c *CasProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	return nil, nil
}

// NotifyPods 异步更新pod的状态。
func (c *CasProvider) NotifyPods(ctx context.Context, notifyStatus func(*corev1.Pod)) {

}

// GetContainerLogs 获取容器日志
func (c *CasProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	return nil, nil
}

// RunInContainer 执行pod中的容器逻辑
func (c *CasProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return nil
}

// ConfigureNode 初始化自定义node节点信息
func (c *CasProvider) ConfigureNode(ctx context.Context, node *corev1.Node) {
	nodes, err := c.clientCache.nodeLister.List(labels.Everything())
	if err != nil {
		return
	}

	nodeResource := common.NewResource()

	for _, n := range nodes {
		if n.Spec.Unschedulable {
			continue
		}
		if !checkNodeStatusReady(n) {
			klog.Infof("Node %v not ready", node.Name)
			continue
		}
		nc := common.ConvertResource(n.Status.Capacity)
		nodeResource.Add(nc)
	}
	nodeResource.SetCapacityToNode(node)
	node.Status.NodeInfo.OperatingSystem = "linux"
	node.Status.NodeInfo.Architecture = "amd64"
	node.ObjectMeta.Labels[corev1.LabelArchStable] = "amd64"
	node.ObjectMeta.Labels[corev1.LabelOSStable] = "linux"
	node.ObjectMeta.Labels[util.LabelOSBeta] = "linux"
	node.Status.Conditions = nodeConditions()
	node.Status.Addresses = []corev1.NodeAddress{
		{
			Type:    corev1.NodeInternalIP,
			Address: "127.0.0.1",
		},
		{
			Type:    corev1.NodeHostName,
			Address: c.nodeName,
		},
	}
	c.providerNode.Node = node
	c.configured = true
	return
}

// Ping tries to connect to client cluster
// implement node.NodeProvider
func (c *CasProvider) Ping(ctx context.Context) error {

	_, err := c.client.Discovery().ServerVersion()
	if err != nil {
		klog.Error("Failed ping")
		return fmt.Errorf("could not list client apiserver statuses: %v", err)
	}
	return nil
}

// NotifyNodeStatus is used to asynchronously monitor the node.
// The passed in callback should be called any time there is a change to the
// node's status.
// This will generally trigger a call to the Kubernetes API server to update
// the status.
//
// NotifyNodeStatus should not block callers.
func (c *CasProvider) NotifyNodeStatus(ctx context.Context, f func(*corev1.Node)) {
	klog.Info("Called NotifyNodeStatus")
	go func() {
		for {
			select {
			case node := <-c.updatedNode:
				klog.Infof("Enqueue updated node %v", node.Name)
				f(node)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// nodeDaemonEndpoints returns NodeDaemonEndpoints for the node status
// within Kubernetes.
func (c *CasProvider) nodeDaemonEndpoints() corev1.NodeDaemonEndpoints {
	return corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			//Port: c.daemonPort,
		},
	}
}

// getResourceFromPods summary the resource already used by pods.
func (c *CasProvider) getResourceFromPods() *common.Resource {
	podResource := common.NewResource()
	/*pods, err := v.clientCache.podLister.List(labels.Everything())
	if err != nil {
		return podResource
	}
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodPending && pod.Spec.NodeName != "" ||
			pod.Status.Phase == corev1.PodRunning {
			nodeName := pod.Spec.NodeName
			node, err := v.clientCache.nodeLister.Get(nodeName)
			if err != nil {
				klog.Infof("get node %v failed err: %v", nodeName, err)
				continue
			}
			if node.Spec.Unschedulable || !checkNodeStatusReady(node) {
				continue
			}
			res := util.GetRequestFromPod(pod)
			res.Pods = resource.MustParse("1")
			podResource.Add(res)
		}
	}*/
	return podResource
}

// getResourceFromPodsByNodeName summary the resource already used by pods according to nodeName
func (c *CasProvider) getResourceFromPodsByNodeName(nodeName string) *common.Resource {
	podResource := common.NewResource()
	/*fieldSelector, err := fields.ParseSelector("spec.nodeName=" + nodeName)
	pods, err := v.client.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(),
		metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		})
	if err != nil {
		return podResource
	}
	for _, pod := range pods.Items {
		if util.IsVirtualPod(&pod) {
			continue
		}
		if pod.Status.Phase == corev1.PodPending ||
			pod.Status.Phase == corev1.PodRunning {
			res := util.GetRequestFromPod(&pod)
			res.Pods = resource.MustParse("1")
			podResource.Add(res)
		}
	}*/
	return podResource
}

// nodeConditions creates a slice of node conditions representing a
// kubelet in perfect health. These four conditions are the ones which virtual-kubelet
// sets as Unknown when a Ping fails.
func nodeConditions() []corev1.NodeCondition {
	return []corev1.NodeCondition{
		{
			Type:               "Ready",
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "kubelet is posting ready status",
		},
		{
			Type:               "MemoryPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientMemory",
			Message:            "kubelet has sufficient memory available",
		},
		{
			Type:               "DiskPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "kubelet has no disk pressure",
		},
		{
			Type:               "PIDPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientPID",
			Message:            "kubelet has sufficient PID available",
		},
	}
}
