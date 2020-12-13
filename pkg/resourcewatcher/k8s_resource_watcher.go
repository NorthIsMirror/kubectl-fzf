package resourcewatcher

import (
	"context"
	"path"
	"strings"
	"time"

	"kubectlfzf/pkg/k8sresources"
	"kubectlfzf/pkg/util"
	"regexp"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchbetav1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	betav1 "k8s.io/api/extensions/v1beta1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	// Import for oidc auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// ResourceWatcher contains rest clients for a given kubernetes context
type ResourceWatcher struct {
	clientset          *kubernetes.Clientset
	namespaces         []string // List of namespaces filtered using excludedNamespaces
	excludedNamespaces []*regexp.Regexp
	cluster            string
	cancelFuncs        []context.CancelFunc
	storeConfig        StoreConfig
}

// WatchConfig provides the configuration to watch a specific kubernetes resource
type WatchConfig struct {
	resourceCtor      func(obj interface{}, config k8sresources.CtorConfig) k8sresources.K8sResource
	header            string
	resourceName      string
	getter            cache.Getter
	runtimeObject     runtime.Object
	hasNamespace      bool
	splitByNamespaces bool
	pollingPeriod     time.Duration
}

// NewResourceWatcher creates a new resource watcher on a given cluster
func NewResourceWatcher(config *restclient.Config,
	storeConfig StoreConfig, excludedNamespaces []string) ResourceWatcher {
	var err error
	resourceWatcher := ResourceWatcher{}
	resourceWatcher.clientset, err = kubernetes.NewForConfig(config)
	util.FatalIf(err)
	resourceWatcher.storeConfig = storeConfig
	resourceWatcher.excludedNamespaces = make([]*regexp.Regexp, len(excludedNamespaces))
	for i, ns := range excludedNamespaces {
		rg, err := regexp.Compile(ns)
		util.FatalIf(err)
		resourceWatcher.excludedNamespaces[i] = rg
	}
	glog.Infof("%d Namespaces will be excluded: %s", len(excludedNamespaces), excludedNamespaces)
	return resourceWatcher
}

// Start begins the watch/poll of a given k8s resource
func (r *ResourceWatcher) Start(parentCtx context.Context, cfg WatchConfig, ctorConfig k8sresources.CtorConfig) error {
	ctx, cancel := context.WithCancel(parentCtx)
	r.cancelFuncs = append(r.cancelFuncs, cancel)

	if cfg.pollingPeriod > 0 {
		store, err := NewK8sStore(ctx, cfg, r.storeConfig, ctorConfig)
		if err != nil {
			return err
		}
		go r.pollResource(ctx, cfg, store)
		return nil
	}

	if cfg.splitByNamespaces {
		glog.Infof("Starting watcher for ns %v, resource %s", r.namespaces, cfg.resourceName)
		store, err := NewK8sStore(ctx, cfg, r.storeConfig, ctorConfig)
		if err != nil {
			return err
		}
		go r.watchResource(ctx, cfg, store, r.namespaces)
		return nil
	}

	store, err := NewK8sStore(ctx, cfg, r.storeConfig, ctorConfig)
	if err != nil {
		return err
	}
	go r.watchResource(ctx, cfg, store, []string{""})
	return nil
}

// Stop closes the watch/poll process of a k8s resource
func (r *ResourceWatcher) Stop() {
	glog.Infof("Stopping %d resource watcher", len(r.cancelFuncs))
	for _, cancel := range r.cancelFuncs {
		cancel()
	}
}

// GetWatchConfigs creates the list of k8s to watch
func (r *ResourceWatcher) GetWatchConfigs(nodePollingPeriod time.Duration, namespacePollingPeriod time.Duration, excludedResources []string) []WatchConfig {
	coreGetter := r.clientset.CoreV1().RESTClient()
	appsGetter := r.clientset.AppsV1().RESTClient()
	autoscalingGetter := r.clientset.AutoscalingV1().RESTClient()
	betaGetter := r.clientset.ExtensionsV1beta1().RESTClient()
	batchGetterV1Beta := r.clientset.BatchV1beta1().RESTClient()
	batchGetter := r.clientset.BatchV1().RESTClient()

	allWatchConfigs := []WatchConfig{
		{k8sresources.NewPodFromRuntime, k8sresources.PodHeader, string(corev1.ResourcePods), coreGetter, &corev1.Pod{}, true, true, 0},
		{k8sresources.NewConfigMapFromRuntime, k8sresources.ConfigMapHeader, "configmaps", coreGetter, &corev1.ConfigMap{}, true, true, 0},
		{k8sresources.NewServiceFromRuntime, k8sresources.ServiceHeader, string(corev1.ResourceServices), coreGetter, &corev1.Service{}, true, false, 0},
		{k8sresources.NewServiceAccountFromRuntime, k8sresources.ServiceAccountHeader, "serviceaccounts", coreGetter, &corev1.Service{}, true, false, 0},
		{k8sresources.NewReplicaSetFromRuntime, k8sresources.ReplicaSetHeader, "replicasets", appsGetter, &appsv1.ReplicaSet{}, true, false, 0},
		{k8sresources.NewDaemonSetFromRuntime, k8sresources.DaemonSetHeader, "daemonsets", appsGetter, &appsv1.DaemonSet{}, true, false, 0},
		{k8sresources.NewSecretFromRuntime, k8sresources.SecretHeader, "secrets", coreGetter, &corev1.Secret{}, true, false, 0},
		{k8sresources.NewStatefulSetFromRuntime, k8sresources.StatefulSetHeader, "statefulsets", appsGetter, &appsv1.StatefulSet{}, true, false, 0},
		{k8sresources.NewDeploymentFromRuntime, k8sresources.DeploymentHeader, "deployments", appsGetter, &appsv1.Deployment{}, true, false, 0},
		{k8sresources.NewEndpointsFromRuntime, k8sresources.EndpointsHeader, "endpoints", coreGetter, &corev1.Endpoints{}, true, false, 0},
		{k8sresources.NewIngressFromRuntime, k8sresources.IngressHeader, "ingresses", betaGetter, &betav1.Ingress{}, true, false, 0},
		{k8sresources.NewCronJobFromRuntime, k8sresources.CronJobHeader, "cronjobs", batchGetterV1Beta, &batchbetav1.CronJob{}, true, false, 0},
		{k8sresources.NewJobFromRuntime, k8sresources.JobHeader, "jobs", batchGetter, &batchv1.Job{}, true, false, 0},
		{k8sresources.NewHpaFromRuntime, k8sresources.HpaHeader, "horizontalpodautoscalers", autoscalingGetter, &autoscalingv1.HorizontalPodAutoscaler{}, true, false, 0},
		{k8sresources.NewPersistentVolumeFromRuntime, k8sresources.PersistentVolumeHeader, "persistentvolumes", coreGetter, &corev1.PersistentVolume{}, false, false, 0},
		{k8sresources.NewPersistentVolumeClaimFromRuntime, k8sresources.PersistentVolumeClaimHeader, string(corev1.ResourcePersistentVolumeClaims), coreGetter, &corev1.PersistentVolumeClaim{}, true, false, 0},
		{k8sresources.NewNodeFromRuntime, k8sresources.NodeHeader, "nodes", coreGetter, &corev1.Node{}, false, false, nodePollingPeriod},
		{k8sresources.NewNamespaceFromRuntime, k8sresources.NamespaceHeader, "namespaces", coreGetter, &corev1.Namespace{}, false, false, namespacePollingPeriod},
	}
	watchConfigs := []WatchConfig{}
	excludedResourcesSet := util.StringSliceToSet(excludedResources)
	glog.Infof("%d Resources will be excluded: %s", len(excludedResources), excludedResources)
	for _, w := range allWatchConfigs {
		if _, ok := excludedResourcesSet[w.resourceName]; ok {
			continue
		}
		watchConfigs = append(watchConfigs, w)
	}

	return watchConfigs
}

func (r *ResourceWatcher) doPoll(watchlist *cache.ListWatch, k8sStore *K8sStore) {
	obj, err := watchlist.List(metav1.ListOptions{})
	if err != nil {
		glog.Warningf("Error on listing %s: %v", k8sStore.resourceName, err)
	}
	lst, err := apimeta.ExtractList(obj)
	if err != nil {
		glog.Warningf("Error extracting list %s: %v", k8sStore.resourceName, err)
	}
	k8sStore.AddResourceList(lst)
}

func (r *ResourceWatcher) FetchNamespaces(ctx context.Context) error {
	namespaces, err := r.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, namespace := range namespaces.Items {
		namespaceName := namespace.GetName()
		if util.IsStringExcluded(namespaceName, r.excludedNamespaces) {
			continue
		}
		r.namespaces = append(r.namespaces, namespaceName)
	}
	glog.Infof("Fetched %d namespaces", len(r.namespaces))
	return nil
}

// DumpAPIResources dumps api resources file
func (r *ResourceWatcher) DumpAPIResources() error {
	resourceName := "apiresources"
	destDir := path.Join(r.storeConfig.CacheDir, r.storeConfig.ClusterDir)
	err := util.WriteStringToFile(k8sresources.APIResourceHeader, destDir, resourceName, "header")
	if err != nil {
		return err
	}

	var res strings.Builder
	resourceLists, _ := r.clientset.Discovery().ServerPreferredResources()
	if err != nil {
		return err
	}
	for _, resourceList := range resourceLists {
		for _, apiResource := range resourceList.APIResources {
			a := k8sresources.APIResource{}
			a.FromAPIResource(apiResource, resourceList)
			_, err := res.WriteString(a.ToString())
			if err != nil {
				return err
			}
		}
	}

	err = util.WriteStringToFile(res.String(), destDir, resourceName, "resource")
	return err
}

func (r *ResourceWatcher) getWatchList(cfg WatchConfig, k8sStore *K8sStore, namespace string) *cache.ListWatch {
	optionsModifier := func(options *metav1.ListOptions) {
		options.FieldSelector = fields.Everything().String()
		options.ResourceVersion = "0"
	}
	watchlist := cache.NewFilteredListWatchFromClient(cfg.getter,
		k8sStore.resourceName, namespace, optionsModifier)
	return watchlist
}

func (r *ResourceWatcher) pollResource(ctx context.Context,
	cfg WatchConfig, k8sStore *K8sStore) {
	glog.V(4).Infof("Start poller for %s", k8sStore.resourceName)
	watchlist := r.getWatchList(cfg, k8sStore, "")

	r.doPoll(watchlist, k8sStore)
	ticker := time.NewTicker(cfg.pollingPeriod)
	for {
		select {
		case <-ctx.Done():
			glog.Infof("Exiting poll of %s", k8sStore.resourceName)
			return
		case <-ticker.C:
			r.doPoll(watchlist, k8sStore)
		}
	}
}

func (r *ResourceWatcher) startWatch(cfg WatchConfig,
	k8sStore *K8sStore, namespace string, stop chan struct{}) {
	watchlist := r.getWatchList(cfg, k8sStore, namespace)
	_, controller := cache.NewInformer(
		watchlist, cfg.runtimeObject, time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    k8sStore.AddResource,
			DeleteFunc: k8sStore.DeleteResource,
			UpdateFunc: k8sStore.UpdateResource,
		},
	)

	controller.Run(stop)
}

func (r *ResourceWatcher) watchResource(ctx context.Context,
	cfg WatchConfig, k8sStore *K8sStore, namespaces []string) {
	glog.V(4).Infof("Start watch for %s on namespace %s", k8sStore.resourceName, namespaces)

	stop := make(chan struct{})
	for _, ns := range namespaces {
		go r.startWatch(cfg, k8sStore, ns, stop)
	}

	<-ctx.Done()
	glog.Infof("Exiting watch of %s namespace %s", k8sStore.resourceName, namespaces)
	close(stop)
}
