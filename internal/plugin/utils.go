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
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// LoadConfigMap reads the configmap and stores the filtering rules in memory.
func LoadConfigMap(configMapData map[string]Resource, namespace, backupName string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("error creating in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %v", err)
	}

	// Use backupName as the name of the configmap
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
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
		// Key: group/version/resource/name/namespace (namespace is optional)
		key := fmt.Sprintf("%s/%s/%s/%s", resource.Group, resource.Version, resource.Resource, resource.Name)
		if resource.Namespace != "" {
			key = fmt.Sprintf("%s/%s", key, resource.Namespace)
		}

		configMapData[key] = resource
	}

	return nil
}
