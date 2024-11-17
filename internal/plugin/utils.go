package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Resource represents a resource entry in the ConfigMap
type Resource struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Kind      string `json:"kind,omitempty"`
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
	configMapName := fmt.Sprintf("%s%s", backupRestoreName, suffix)

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
func IsResourceMatch(resource Resource, gvrGroup, gvrVersion, gvrKind, name, namespace string, log logrus.FieldLogger) bool {
	// Convert all fields to lowercase for comparison
	resourceName := strings.ToLower(resource.Name)
	inputName := strings.ToLower(name)
	resourceNamespace := strings.ToLower(resource.Namespace)
	inputNamespace := strings.ToLower(namespace)
	resourceGroup := strings.ToLower(resource.Group)
	inputGroup := strings.ToLower(gvrGroup)
	resourceVersion := strings.ToLower(resource.Version)
	inputVersion := strings.ToLower(gvrVersion)
	resourceKind := strings.ToLower(resource.Kind)
	inputKind := strings.ToLower(gvrKind)

	// Log the values being compared for name
	log.Infof("Comparing resource name: '%s' with input name: '%s'", resourceName, inputName)
	if resourceName != inputName {
		log.Warnf("Name mismatch: resource name '%s' does not match input name '%s'", resourceName, inputName)
		return false
	}

	// Optionally check the namespace (if present) and log the comparison
	if resource.Namespace != "" {
		log.Infof("Comparing resource namespace: '%s' with input namespace: '%s'", resourceNamespace, inputNamespace)
		if resourceNamespace != inputNamespace {
			log.Warnf("Namespace mismatch: resource namespace '%s' does not match input namespace '%s'", resourceNamespace, inputNamespace)
			return false
		}
	}

	// Optionally check GVR (group/version/kind) but all 3 must match at once if provided
	if resource.Group != "" || resource.Version != "" || resource.Kind != "" {
		// Log the values being compared for GVR
		log.Infof("Comparing resource group: '%s' with input group: '%s'", resourceGroup, inputGroup)
		log.Infof("Comparing resource version: '%s' with input version: '%s'", resourceVersion, inputVersion)
		log.Infof("Comparing resource kind: '%s' with input kind: '%s'", resourceKind, inputKind)

		// If any part of the GVR is provided, ensure all 3 match at the same time
		if resourceGroup != inputGroup || resourceVersion != inputVersion || resourceKind != inputKind {
			log.Warnf("GVR mismatch: resource GVR '%s/%s/%s' does not match input GVR '%s/%s/%s'",
				resourceGroup, resourceVersion, resourceKind, inputGroup, inputVersion, inputKind)
			return false
		}
	}

	// If all checks pass, log success and return true
	log.Info("Resource matches all criteria")
	return true
}
