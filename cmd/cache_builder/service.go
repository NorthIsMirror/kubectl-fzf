package main

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
)

// Service is the summary of a kubernetes service
type Service struct {
	ResourceMeta
	serviceType string
	clusterIP   string
	ports       []string
	selectors   []string
}

// FromRuntime builds object from the informer's result
func (s *Service) FromRuntime(obj interface{}) {
	service := obj.(*corev1.Service)
	glog.V(19).Infof("Reading meta %#v", service)
	s.FromObjectMeta(service.ObjectMeta)
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
	s.selectors = JoinStringMap(service.Spec.Selector, ExcludedLabels, "=")
}

// HasChanged returns true if the resource's dump needs to be updated
func (s *Service) HasChanged(k K8sResource) bool {
	oldService := k.(*Service)
	return (StringSlicesEqual(s.ports, oldService.ports) ||
		StringSlicesEqual(s.selectors, oldService.selectors) ||
		StringMapsEqual(s.labels, oldService.labels))
}

// Header generates the csv header for the resource
func (s *Service) Header() string {
	return "Namespace Name Type ClusterIp Ports Selector Age Labels\n"
}

// ToString serializes the object to strings
func (s *Service) ToString() string {
	portList := JoinSlicesOrNone(s.ports, ",")
	selectorList := JoinSlicesOrNone(s.selectors, ",")
	line := strings.Join([]string{s.namespace,
		s.name,
		s.serviceType,
		s.clusterIP,
		portList,
		selectorList,
		s.resourceAge(),
		s.labelsString(),
	}, " ")
	return fmt.Sprintf("%s\n", line)
}