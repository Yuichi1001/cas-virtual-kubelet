package providers

import (
	"context"
	"fmt"
	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	informerv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"reflect"
)

const (
	PodLogRoot      = "/var/log/vk-cri/"
	PodVolRoot      = "/run/vk-cri/volumes/"
	PodLogRootPerms = 0755
	PodVolRootPerms = 0755
)

// clientCache wraps the lister of client cluster
type clientCache struct {
	nodeLister v1.NodeLister
}

// CasProvider 实现virtual-kubelet对象
type CasProvider struct {
	// options 配置
	options *common.ProviderConfig
	// PodManager 管理pods状态管理
	PodManager *PodManager
	// podLogRoot 存放容器日志目录
	podLogRoot string
	// podVolRoot 存放容器挂载目录
	podVolRoot string
	// nodeName 节点名称，初始化时必须指定
	nodeName string
	// checkPeriod 检查定时周期
	checkPeriod int64
	notifyC     chan struct{}
	// 上报的回调方法，主要把本节点中的pod status放入工作队列
	notifyStatus func(*corev1.Pod)

	client       *kubernetes.Clientset
	configured   bool
	providerNode *common.ProviderNode
	updatedNode  chan *corev1.Node
	clientCache  clientCache
}

// 是否实现下列两种接口，这是vk组件必须实现的两个接口。
var _ node.PodLifecycleHandler = &CasProvider{}
var _ node.PodNotifier = &CasProvider{}

func NewCriProvider(ctx context.Context, options *common.ProviderConfig) *CasProvider {
	config, err := clientcmd.BuildConfigFromFlags("", options.ClientConfig)
	if err != nil {
		fmt.Println("BuildConfigFromFlags_err:", err)
		return nil
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("newforconfig_err:", err)
		return nil
	}

	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	nodeInformer := informerFactory.Core().V1().Nodes()

	provider := &CasProvider{
		options:    options,
		podLogRoot: PodLogRoot,
		podVolRoot: PodVolRoot,
		PodManager: NewPodManager(),
		nodeName:   options.NodeName,
		notifyC:    make(chan struct{}),
		client:     clientset,
		clientCache: clientCache{
			nodeLister: nodeInformer.Lister(),
		},
		updatedNode:  make(chan *corev1.Node, 100),
		providerNode: &common.ProviderNode{},
	}

	provider.buildNodeInformer(nodeInformer)

	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	// 初始化时先创建目录
	err = os.MkdirAll(provider.podLogRoot, PodLogRootPerms)
	if err != nil {
		return nil
	}
	err = os.MkdirAll(provider.podVolRoot, PodVolRootPerms)
	if err != nil {
		return nil
	}

	return provider
}

func (c *CasProvider) buildNodeInformer(nodeInformer informerv1.NodeInformer) {

	nodeInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			//添加节点时无需addresource，因为新加入的节点从notready变为ready时会触发update，在updatefunc里面进行addresource
			AddFunc: func(obj interface{}) {
				/*if !c.configured {
					return
				}
				nodeCopy := c.providerNode.DeepCopy()
				addNode := obj.(*corev1.Node).DeepCopy()
				fmt.Println(addNode.Name)
				toAdd := common.ConvertResource(addNode.Status.Capacity)
				if err := c.providerNode.AddResource(toAdd); err != nil {
					return
				}
				// resource we did not add when ConfigureNode should sub
				//p.providerNode.SubResource(p.getResourceFromPodsByNodeName(addNode.Name))
				copy := c.providerNode.DeepCopy()
				if !reflect.DeepEqual(nodeCopy, copy) {
					c.updatedNode <- copy
				}*/
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if !c.configured {
					return
				}
				old, ok1 := oldObj.(*corev1.Node)
				new, ok2 := newObj.(*corev1.Node)
				oldCopy := old.DeepCopy()
				newCopy := new.DeepCopy()
				if !ok1 || !ok2 {
					return
				}
				c.updateVKCapacityFromNode(oldCopy, newCopy)
			},
			DeleteFunc: func(obj interface{}) {
				if !c.configured {
					return
				}
				deleteNode := obj.(*corev1.Node).DeepCopy()
				if deleteNode.Spec.Unschedulable || !checkNodeStatusReady(deleteNode) {
					return
				}
				nodeCopy := c.providerNode.DeepCopy()
				toRemove := common.ConvertResource(deleteNode.Status.Capacity)
				if err := c.providerNode.SubResource(toRemove); err != nil {
					return
				}
				// resource we did not add when ConfigureNode should add
				//p.providerNode.AddResource(p.getResourceFromPodsByNodeName(deleteNode.Name))
				copy := c.providerNode.DeepCopy()
				if !reflect.DeepEqual(nodeCopy, copy) {
					c.updatedNode <- copy
				}
			},
		},
	)
}

func checkNodeStatusReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type != corev1.NodeReady {
			continue
		}
		if condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func compareNodeStatusReady(old, new *corev1.Node) (bool, bool) {
	return checkNodeStatusReady(old), checkNodeStatusReady(new)
}

/*
// getResourceFromPodsByNodeName summary the resource already used by pods according to nodeName
func (v *CasProvider) getResourceFromPodsByNodeName(nodeName string) *common.Resource {
	podResource := common.NewResource()
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + nodeName)
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
	}
	return podResource
}*/

func (c *CasProvider) updateVKCapacityFromNode(old, new *corev1.Node) {
	oldStatus, newStatus := compareNodeStatusReady(old, new)
	if !oldStatus && !newStatus {
		return
	}
	toRemove := common.ConvertResource(old.Status.Capacity)
	toAdd := common.ConvertResource(new.Status.Capacity)
	nodeCopy := c.providerNode.DeepCopy()

	if c.providerNode.Node == nil {
		return
	} else if old.Spec.Unschedulable && !new.Spec.Unschedulable || newStatus && !oldStatus {
		c.providerNode.AddResource(toAdd)
		//v.providerNode.SubResource(v.getResourceFromPodsByNodeName(old.Name))
	} else if !old.Spec.Unschedulable && new.Spec.Unschedulable || oldStatus && !newStatus {
		//v.providerNode.AddResource(v.getResourceFromPodsByNodeName(old.Name))
		c.providerNode.SubResource(toRemove)

	} else if !reflect.DeepEqual(old.Status.Allocatable, new.Status.Allocatable) ||
		!reflect.DeepEqual(old.Status.Capacity, new.Status.Capacity) {
		c.providerNode.AddResource(toAdd)
		c.providerNode.SubResource(toRemove)
	}
	copy := c.providerNode.DeepCopy()
	if !reflect.DeepEqual(nodeCopy, copy) {
		c.updatedNode <- copy
	}
}

// NotifyPods 异步更新pod的状态。
// 需要实现 node.PodNotifier 对象
func (c *CasProvider) NotifyPods(ctx context.Context, notifyStatus func(*corev1.Pod)) {

}
