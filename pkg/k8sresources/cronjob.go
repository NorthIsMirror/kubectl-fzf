package k8sresources

import (
	"kubectlfzf/pkg/util"
	"strings"

	"github.com/golang/glog"
	v1 "k8s.io/api/batch/v1"
)

// CronJobHeader is the headers for cronjob files
const CronJobHeader = "Cluster Namespace Name Schedule LastSchedule Containers Age Labels\n"

// CronJob is the summary of a kubernetes cronJob
type CronJob struct {
	ResourceMeta
	schedule     string
	lastSchedule string
	containers   []string
}

// NewCronJobFromRuntime builds a cronJob from informer result
func NewCronJobFromRuntime(obj interface{}, config CtorConfig) K8sResource {
	c := &CronJob{}
	c.FromRuntime(obj, config)
	return c
}

// FromRuntime builds object from the informer's result
func (c *CronJob) FromRuntime(obj interface{}, config CtorConfig) {
	cronJob := obj.(*v1.CronJob)
	glog.V(19).Infof("Reading meta %#v", cronJob)
	c.FromObjectMeta(cronJob.ObjectMeta, config)
	c.schedule = strings.ReplaceAll(cronJob.Spec.Schedule, " ", "_")
	c.lastSchedule = ""
	if cronJob.Status.LastScheduleTime != nil {
		c.lastSchedule = util.TimeToAge(cronJob.Status.LastScheduleTime.Time)
	}

	spec := cronJob.Spec.JobTemplate.Spec.Template.Spec
	containers := spec.Containers
	containers = append(containers, spec.InitContainers...)
	c.containers = make([]string, len(containers))
	for k, v := range containers {
		c.containers[k] = v.Name
	}
}

// HasChanged returns true if the resource's dump needs to be updated
func (c *CronJob) HasChanged(k K8sResource) bool {
	return true
}
