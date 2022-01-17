package k8sresources

import (
	"kubectlfzf/pkg/util"

	betav1 "k8s.io/api/extensions/v1beta1"
)

const IngressHeader = "Cluster Namespace Name Address Age Labels\n"

// Ingress is the summary of a kubernetes ingress
type Ingress struct {
	ResourceMeta
	Address []string
}

// NewIngressFromRuntime builds a pod from informer result
func NewIngressFromRuntime(obj interface{}, config CtorConfig) K8sResource {
	p := &Ingress{}
	p.FromRuntime(obj, config)
	return p
}

// FromRuntime builds object from the informer's result
func (ingress *Ingress) FromRuntime(obj interface{}, config CtorConfig) {
	ingressFromRuntime := obj.(*betav1.Ingress)
	ingress.FromObjectMeta(ingressFromRuntime.ObjectMeta, config)
	for _, lb := range ingressFromRuntime.Status.LoadBalancer.Ingress {
		ingress.Address = append(ingress.Address, lb.Hostname)
	}
}

// HasChanged returns true if the resource's dump needs to be updated
func (ingress *Ingress) HasChanged(k K8sResource) bool {
	return true
}

// ToString serializes the object to strings
func (ingress *Ingress) ToString() string {
	addressList := util.JoinSlicesOrNone(ingress.Address, ",")
	lst := []string{
		ingress.Cluster,
		ingress.Namespace,
		ingress.Name,
		addressList,
		ingress.resourceAge(),
		ingress.labelsString(),
	}
	return util.DumpLine(lst)
}
