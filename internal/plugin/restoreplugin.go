package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type RestorePlugin struct {
	log           logrus.FieldLogger
	configMapData map[string]bool
	configLoaded  bool
}

// NewRestorePlugin instantiates a RestorePlugin.
func NewRestorePlugin(log logrus.FieldLogger) *RestorePlugin {
	return &RestorePlugin{log: log}
}

// LoadConfigMap reads the configmap and stores the filtering rules in memory.
func (p *RestorePlugin) LoadConfigMap(namespace, name string) error {
	if p.configLoaded {
		return nil
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("error creating in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %v", err)
	}

	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting configmap: %v", err)
	}

	p.configMapData = make(map[string]bool)

	// Assuming that the ConfigMap has a simple format like:
	// resources: group/version/resource/name[/namespace]
	resources := cm.Data["resources"]
	for _, line := range strings.Split(resources, "\n") {
		if line != "" {
			p.configMapData[line] = true
		}
	}

	p.configLoaded = true
	return nil
}

// ExecuteRestore is the function that performs restore logic based on ConfigMap filtering.
func (p *RestorePlugin) Execute(item runtime.Unstructured, restore *v1.Restore) (*velero.RestoreItemActionExecuteOutput, error) {
	// Load the configmap on the first execution
	if !p.configLoaded {
		if err := p.LoadConfigMap("velero", "resource-filter"); err != nil {
			return nil, fmt.Errorf("error loading configmap: %v", err)
		}
	}

	// Extract GVR, Name, and optionally Namespace
	gvr := item.GetObjectKind().GroupVersionKind()
	metadata, err := meta.Accessor(item)
	if err != nil {
		return nil, fmt.Errorf("error accessing metadata: %v", err)
	}
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// Build the key to look up in the configmap (group/version/resource/name[/namespace])
	key := fmt.Sprintf("%s/%s/%s/%s", gvr.Group, gvr.Version, gvr.Kind, name)
	if namespace != "" {
		key = fmt.Sprintf("%s/%s", key, namespace)
	}

	// Check if the resource should be restored
	if p.configMapData[key] {
		p.log.Infof("Restoring resource: %s", key)
		return velero.NewRestoreItemActionExecuteOutput(item), nil
	}

	p.log.Infof("Skipping resource: %s", key)
	return &velero.RestoreItemActionExecuteOutput{
		SkipRestore: true,
	}, nil
}
