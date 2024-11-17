package plugin

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
)

// RestoreFilterPlugin is a restore item action plugin for Velero.
type RestoreFilterPlugin struct {
	log               logrus.FieldLogger
	configMapData     map[string]Resource
	configLoaded      bool
	configMapNotFound bool
}

// NewRestoreFilterPlugin instantiates a RestorePlugin.
func NewRestoreFilterPlugin(log logrus.FieldLogger) *RestoreFilterPlugin {
	log.Info("Entering NewRestorePlugin function.")
	defer log.Info("Exiting NewRestorePlugin function.")

	return &RestoreFilterPlugin{
		log:               log,
		configMapData:     make(map[string]Resource),
		configMapNotFound: false,
	}
}

// AppliesTo returns information about which resources this action should be invoked for.
func (p *RestoreFilterPlugin) AppliesTo() (velero.ResourceSelector, error) {
	p.log.Info("Entering AppliesTo function.")
	defer p.log.Info("Exiting AppliesTo function.")

	// Log that AppliesTo is invoked but returns everything
	p.log.Info("AppliesTo will apply to all resources except namespaces.")
	return velero.ResourceSelector{
		ExcludedResources: []string{"namespaces"}, // Exclude namespaces from filtering
	}, nil
}

// Execute is the function that performs restore logic based on ConfigMap filtering.
func (p *RestoreFilterPlugin) Execute(inputPayload *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	item := inputPayload.Item
	restore := inputPayload.Restore
	p.log.Info(" ==> Entering Execute function.")
	defer p.log.Info(" ==> Exiting Execute function.")

	// If the ConfigMap was not found, allow restore without filtering
	if p.configMapNotFound {
		p.log.Info(" ==> ConfigMap not found earlier. Allowing restore without filtering.")
		return velero.NewRestoreItemActionExecuteOutput(item), nil
	}

	// Load the configmap on the first execution
	if !p.configLoaded {
		p.log.Infof(" ==> Loading ConfigMap for restore: %s-r-r-f in namespace: %s", restore.Name, restore.Namespace)
		err := LoadConfigMap(p.configMapData, restore.Namespace, restore.Name, "-r-r-f")
		if err != nil {
			p.log.Warnf(" ==> ConfigMap not found or error loading configmap: %v. Allowing all resources to be restored.", err)
			p.configMapNotFound = true
			return velero.NewRestoreItemActionExecuteOutput(item), nil // Allow all resources to be restored if the ConfigMap is not found
		}
		p.configLoaded = true
	}

	// Extract GVR, Name, and Namespace
	gvr := item.GetObjectKind().GroupVersionKind()
	metadata, err := meta.Accessor(item)
	if err != nil {
		p.log.Errorf(" ==> Error accessing metadata for item: %v", err)
		return nil, fmt.Errorf("error accessing metadata: %v", err)
	}
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// Log the GVR and name of the resource being processed
	p.log.Infof(" ==> Processing resource: GVR = %s/%s/%s, Name = %s, Namespace = %s", gvr.Group, gvr.Version, gvr.Kind, name, namespace)

	/*
		We need to check if the namespace of the item being restored has been mapped to a different namespace
		by a previous plugin using Velero’s namespace mapping feature (input.Restore.Spec.NamespaceMapping).
		If the namespace has been mapped, we should use the original namespace from the mapping for checking
		against the ConfigMap rules.
	*/
	originalNamespace := namespace
	if restore.Spec.NamespaceMapping != nil {
		p.log.Infof(" ==> Checking if namespace %s has been mapped in the restore spec.", namespace)
		for orig, mapped := range restore.Spec.NamespaceMapping {
			p.log.Infof(" ==> Checking namespace mapping: original = %s, mapped = %s", orig, mapped)
			if mapped == namespace {
				p.log.Infof(" ==> Namespace %s has been mapped to %s. Using original namespace %s.", mapped, namespace, orig)
				originalNamespace = orig // Use the original namespace if the current one is mapped
				break
			}
		}
	} else {
		p.log.Info(" ==> No namespace mapping found in the restore spec.")
	}

	// Check if the resource should be restored based on the original namespace
	p.log.Infof(" ==> CC Filter Plugin Configmap JSON content is as follows: %v", p.configMapData)
	p.log.Infof(" ==> Checking if resource: %s in original namespace: %s should be restored.", name, originalNamespace)

	if resource, exists := p.configMapData[name]; exists {
		if IsResourceMatch(resource, gvr.Group, gvr.Version, gvr.Kind, name, originalNamespace, p.log) {
			p.log.Infof(" ==> Resource GVK = %s/%s/%s, Name = %s, Namespace %s matches the ConfigMap criteria and will be restored.", gvr.Group, gvr.Version, gvr.Kind, name, originalNamespace)
			return velero.NewRestoreItemActionExecuteOutput(item), nil
		}
	}

	// Log that the resource is being skipped
	p.log.Infof(" ==> Resource GVK = %s/%s/%s, Name = %s, Namespace %s does not match the ConfigMap criteria and will be skipped.", gvr.Group, gvr.Version, gvr.Kind, name, originalNamespace)
	return &velero.RestoreItemActionExecuteOutput{
		SkipRestore: true,
	}, nil
}
