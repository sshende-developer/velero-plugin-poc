package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Resource represents a resource entry in the ConfigMap
type Resource struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// LoadConfigMap reads the configmap and stores the filtering rules in memory.
// The configmap name follows the format <name of the backup/restore>-b-r-f or -r-r-f
func LoadConfigMap(configMapData map[string]Resource, namespace, backupRestoreName, suffix string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("error creating in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %v", err)
	}

	// Create the configmap name based on backup or restore and suffix
	configMapName := fmt.Sprintf("%s-%s", backupRestoreName, suffix)

	// Use the dynamically constructed ConfigMap name
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting configmap: %v", err)
	}

	// Parse the resources from the ConfigMap
	resources := cm.Data["resources"]
	var resourceList []Resource
	err = json.Unmarshal([]byte(resources), &resourceList)
	if err != nil {
		return fmt.Errorf("error unmarshalling resources from configmap: %v", err)
	}

	// Populate the configMapData with the parsed resources
	for _, resource := range resourceList {
		configMapData[resource.Name] = resource
	}

	return nil
}

// IsResourceMatch checks if a resource matches the filtering rule from the ConfigMap.
// It checks against name (mandatory) and optionally namespace and GVR.
// All of G, V, and R (Group, Version, Resource) must match together if provided.
func IsResourceMatch(resource Resource, gvrGroup, gvrVersion, gvrKind, name, namespace string) bool {
	// First, check if the name matches (name is mandatory)
	if resource.Name != name {
		return false
	}

	// Optionally check the namespace (if present)
	if resource.Namespace != "" && resource.Namespace != namespace {
		return false
	}

	// Optionally check GVR (group/version/resource), but all 3 must match at once if given.
	if resource.Group != "" || resource.Version != "" || resource.Resource != "" {
		// If any part of the GVR is provided, ensure all 3 match at the same time.
		if resource.Group != gvrGroup || resource.Version != gvrVersion || resource.Resource != gvrKind {
			return false
		}
	}

	// If all checks pass, the resource matches
	return true
}
