package k8sresources

import (
	"strconv"

	"kubectlfzf/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
)

const ReplicaSetHeader = "Cluster Namespace Name Replicas AvailableReplicas ReadyReplicas Selector Age Labels\n"

// ReplicaSet is the summary of a kubernetes replicaSet
type ReplicaSet struct {
	ResourceMeta
	replicas          string
	readyReplicas     string
	availableReplicas string
	selectors         []string
}

// NewReplicaSetFromRuntime builds a k8sresource from informer result
func NewReplicaSetFromRuntime(obj interface{}, config CtorConfig) K8sResource {
	p := &ReplicaSet{}
	p.FromRuntime(obj, config)
	return p
}

// FromRuntime builds object from the informer's result
func (r *ReplicaSet) FromRuntime(obj interface{}, config CtorConfig) {
	replicaSet := obj.(*appsv1.ReplicaSet)
	r.FromObjectMeta(replicaSet.ObjectMeta, config)
	r.replicas = strconv.Itoa(int(replicaSet.Status.Replicas))
	r.readyReplicas = strconv.Itoa(int(replicaSet.Status.ReadyReplicas))
	r.availableReplicas = strconv.Itoa(int(replicaSet.Status.AvailableReplicas))
	r.selectors = util.JoinStringMap(replicaSet.Spec.Selector.MatchLabels,
		ExcludedLabels, "=")
}

// HasChanged returns true if the resource'r dump needs to be updated
func (r *ReplicaSet) HasChanged(k K8sResource) bool {
	oldRs := k.(*ReplicaSet)
	return (r.replicas != oldRs.replicas ||
		r.readyReplicas != oldRs.readyReplicas ||
		r.availableReplicas != oldRs.availableReplicas ||
		util.StringSlicesEqual(r.selectors, oldRs.selectors) ||
		util.StringMapsEqual(r.Labels, oldRs.Labels))
}
