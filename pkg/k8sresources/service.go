package k8sresources

import (
	"fmt"

	"kubectlfzf/pkg/util"

	corev1 "k8s.io/api/core/v1"
)

const ServiceHeader = "Cluster Namespace Name Type ClusterIp Ports Selector Age Labels\n"

// Service is the summary of a kubernetes service
type Service struct {
	ResourceMeta
	serviceType string
	clusterIP   string
	ports       []string
	selectors   []string
}

// NewServiceFromRuntime builds a pod from informer result
func NewServiceFromRuntime(obj interface{}, config CtorConfig) K8sResource {
	s := &Service{}
	s.FromRuntime(obj, config)
	return s
}

// FromRuntime builds object from the informer's result
func (s *Service) FromRuntime(obj interface{}, config CtorConfig) {
	service := obj.(*corev1.Service)
	s.FromObjectMeta(service.ObjectMeta, config)
	s.serviceType = string(service.Spec.Type)
	s.clusterIP = service.Spec.ClusterIP
	s.ports = make([]string, len(service.Spec.Ports))
	for k, v := range service.Spec.Ports {
		if v.NodePort > 0 {
			s.ports[k] = fmt.Sprintf("%s:%d/%d", v.Name, v.Port, v.NodePort)
		} else {
			s.ports[k] = fmt.Sprintf("%s:%d", v.Name, v.Port)
		}
	}
	s.selectors = util.JoinStringMap(service.Spec.Selector, ExcludedLabels, "=")
}

// HasChanged returns true if the resource's dump needs to be updated
func (s *Service) HasChanged(k K8sResource) bool {
	oldService := k.(*Service)
	return (util.StringSlicesEqual(s.ports, oldService.ports) ||
		util.StringSlicesEqual(s.selectors, oldService.selectors) ||
		util.StringMapsEqual(s.Labels, oldService.Labels))
}
