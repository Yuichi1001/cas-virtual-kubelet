package util

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// GlobalLabel make object global
	GlobalLabel = "global"
	// SelectorKey is the key of ClusterSelector
	SelectorKey = "clusterSelector"
	// SelectedNodeKey is the node selected by a scheduler
	SelectedNodeKey = "volume.kubernetes.io/selected-node"
	// HostNameKey is the label of HostNameKey
	HostNameKey = "kubernetes.io/hostname"
	// BetaHostNameKey is the label of HostNameKey
	BetaHostNameKey = "beta.kubernetes.io/hostname"
	// LabelOSBeta is the label of os
	LabelOSBeta = "beta.kubernetes.io/os"
	// VirtualPodLabel is the label of virtual pod
	VirtualPodLabel = "virtual-pod"
	// VirtualKubeletLabel is the label of virtual kubelet
	VirtualKubeletLabel = "virtual-kubelet"
	// TrippedLabels is the label of tripped labels
	TrippedLabels = "tripped-labels"
	// ClusterID marks the id of a cluster
	ClusterID = "clusterID"
	// NodeType is define the node type key
	NodeType = "type"
	// BatchPodLabel is the label of batch pod
	BatchPodLabel = "pod-group.scheduling.sigs.k8s.io"
	// TaintNodeNotReady will be added when node is not ready
	// and feature-gate for TaintBasedEvictions flag is enabled,
	// and removed when node becomes ready.
	TaintNodeNotReady = "node.kubernetes.io/not-ready"

	// TaintNodeUnreachable will be added when node becomes unreachable
	// (corresponding to NodeReady status ConditionUnknown)
	// and feature-gate for TaintBasedEvictions flag is enabled,
	// and removed when node becomes reachable (NodeReady status ConditionTrue).
	TaintNodeUnreachable = "node.kubernetes.io/unreachable"
	// CreatedbyDescheduler is used to mark if a pod is re-created by descheduler
	CreatedbyDescheduler = "create-by-descheduler"
	// DescheduleCount is used for recording deschedule count
	DescheduleCount = "sigs.k8s.io/deschedule-count"
)

// IsVirtualNode defines if a node is virtual node
func IsVirtualNode(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	valStr, exist := node.ObjectMeta.Labels[NodeType]
	if !exist {
		return false
	}
	return valStr == VirtualKubeletLabel
}

// IsVirtualPod defines if a pod is virtual pod
func IsVirtualPod(pod *corev1.Pod) bool {
	if pod.Labels != nil && pod.Labels[VirtualPodLabel] == "true" {
		return true
	}
	return false
}
