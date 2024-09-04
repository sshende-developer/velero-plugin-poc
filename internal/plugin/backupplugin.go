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

type BackupPlugin struct {
	log           logrus.FieldLogger
	configMapData map[string]bool
	configLoaded  bool
}

// NewBackupPlugin instantiates a BackupPlugin.
func NewBackupPlugin(log logrus.FieldLogger) *BackupPlugin {
	return &BackupPlugin{log: log}
}

// AppliesTo returns information about which resources this action should be invoked for.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}

// LoadConfigMap reads the configmap and stores the filtering rules in memory.
func (p *BackupPlugin) LoadConfigMap(namespace, name string) error {
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

// Execute is the function that performs backup logic based on ConfigMap filtering.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	// Load the configmap on the first execution
	if !p.configLoaded {
		if err := p.LoadConfigMap("velero", "resource-filter"); err != nil {
			return nil, nil, fmt.Errorf("error loading configmap: %v", err)
		}
	}

	// Extract GVR, Name, and optionally Namespace
	gvr := item.GetObjectKind().GroupVersionKind()
	metadata, err := meta.Accessor(item)
	if err != nil {
		return nil, nil, fmt.Errorf("error accessing metadata: %v", err)
	}
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// Build the key to look up in the configmap (group/version/resource/name[/namespace])
	key := fmt.Sprintf("%s/%s/%s/%s", gvr.Group, gvr.Version, gvr.Kind, name)
	if namespace != "" {
		key = fmt.Sprintf("%s/%s", key, namespace)
	}

	// Check if the resource should be backed up
	if p.configMapData[key] {
		p.log.Infof("Backing up resource: %s", key)
		return item, nil, nil
	}

	p.log.Infof("Skipping resource: %s", key)
	return nil, nil, nil
}
