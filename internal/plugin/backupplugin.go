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
	log               logrus.FieldLogger
	configMapData     map[string]Resource
	configLoaded      bool
	configMapNotFound bool
}

// NewBackupPlugin instantiates a BackupPlugin.
func NewBackupPlugin(log logrus.FieldLogger) *BackupPlugin {
	log.Info("Entering NewBackupPlugin function.")
	defer log.Info("Exiting NewBackupPlugin function.")

	return &BackupPlugin{
		log:               log,
		configMapData:     make(map[string]Resource),
		configMapNotFound: false,
	}
}

// AppliesTo returns information about which resources this action should be invoked for.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	p.log.Info("Entering AppliesTo function.")
	defer p.log.Info("Exiting AppliesTo function.")

	// Log that AppliesTo is invoked but returns everything
	p.log.Info("AppliesTo will apply to all resources except namespaces.")
	return velero.ResourceSelector{
		ExcludedResources: []string{"namespaces"}, // Exclude namespaces from filtering
	}, nil
}

// Execute is the function that performs backup logic based on ConfigMap filtering.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.log.Info("Entering Execute function.")
	defer p.log.Info("Exiting Execute function.")

	// If the ConfigMap was not found, allow backup without filtering
	if p.configMapNotFound {
		p.log.Info("ConfigMap not found earlier. Allowing backup without filtering.")
		return item, nil, nil
	}

	// Load the configmap on the first execution
	if !p.configLoaded {
		p.log.Infof("Loading ConfigMap for backup: %s-b-r-f in namespace: %s", backup.Name, backup.Namespace)
		err := LoadConfigMap(p.configMapData, backup.Namespace, backup.Name, "-b-r-f")
		if err != nil {
			p.log.Warnf("ConfigMap not found or error loading configmap: %v. Allowing all resources to be backed up.", err)
			p.configMapNotFound = true
			return item, nil, nil // Allow all resources to be backed up if the ConfigMap is not found
		}
		p.configLoaded = true
	}

	// Extract GVR, Name, and optionally Namespace
	gvr := item.GetObjectKind().GroupVersionKind()
	metadata, err := meta.Accessor(item)
	if err != nil {
		p.log.Errorf("Error accessing metadata for item: %v", err)
		return nil, nil, fmt.Errorf("error accessing metadata: %v", err)
	}
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// Log the GVR and name of the resource being processed
	p.log.Infof("Processing resource: GVR = %s/%s/%s, Name = %s, Namespace = %s", gvr.Group, gvr.Version, gvr.Kind, name, namespace)

	// Check if the resource should be backed up
	p.log.Infof("Checking if resource: %s in namespace: %s should be backed up.", name, namespace)
	if resource, exists := p.configMapData[name]; exists {
		if IsResourceMatch(resource, gvr.Group, gvr.Version, gvr.Kind, name, namespace) {
			p.log.Infof("Resource GVR = %s/%s/%s, Name = %s, Namespace %s matches the ConfigMap criteria and will be backed up.", gvr.Group, gvr.Version, gvr.Kind, name, namespace)
			return item, nil, nil
		}
	}

	// Log that the resource is being skipped
	p.log.Infof("Resource GVR = %s/%s/%s, Name = %s, Namespace %s does not match the ConfigMap criteria and will be skipped.", gvr.Group, gvr.Version, gvr.Kind, name, namespace)
	return nil, nil, nil
}
