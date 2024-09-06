package plugin

import (
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Velero.
type BackupPlugin struct {
	log           logrus.FieldLogger
	configMapData map[string]Resource
	configLoaded  bool
}

// NewBackupPlugin instantiates a BackupPlugin.
func NewBackupPlugin(log logrus.FieldLogger) *BackupPlugin {
	return &BackupPlugin{
		log:           log,
		configMapData: make(map[string]Resource),
	}
}

// AppliesTo returns information about which resources this action should be invoked for.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}

// Execute is the function that performs backup logic based on ConfigMap filtering.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	// Load the configmap on the first execution
	if !p.configLoaded {
		if err := LoadConfigMap(p.configMapData, backup.Namespace, backup.Name); err != nil {
			return nil, nil, fmt.Errorf("error loading configmap: %v", err)
		}
		p.configLoaded = true
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
	if _, exists := p.configMapData[key]; exists {
		p.log.Infof("Backing up resource: %s", key)
		return item, nil, nil
	}

	p.log.Infof("Skipping resource: %s", key)
	return nil, nil, nil
}
