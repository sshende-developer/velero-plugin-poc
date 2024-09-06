package plugin

import (
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// RestorePlugin is a restore item action plugin for Velero.
type RestorePlugin struct {
	log           logrus.FieldLogger
	configMapData map[string]Resource
	configLoaded  bool
}

// NewRestorePlugin instantiates a RestorePlugin.
func NewRestorePlugin(log logrus.FieldLogger) *RestorePlugin {
	return &RestorePlugin{
		log:           log,
		configMapData: make(map[string]Resource),
	}
}

// AppliesTo returns information about which resources this action should be invoked for.
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}

// ExecuteRestore is the function that performs restore logic based on ConfigMap filtering.
func (p *RestorePlugin) Execute(item runtime.Unstructured, restore *v1.Restore) (*velero.RestoreItemActionExecuteOutput, error) {
	// Load the configmap on the first execution
	if !p.configLoaded {
		if err := LoadConfigMap(p.configMapData, restore.Namespace, restore.Name); err != nil {
			return nil, fmt.Errorf("error loading configmap: %v", err)
		}
		p.configLoaded = true
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
	if _, exists := p.configMapData[key]; exists {
		p.log.Infof("Restoring resource: %s", key)
		return velero.NewRestoreItemActionExecuteOutput(item), nil
	}

	p.log.Infof("Skipping resource: %s", key)
	return &velero.RestoreItemActionExecuteOutput{
		SkipRestore: true,
	}, nil
}
