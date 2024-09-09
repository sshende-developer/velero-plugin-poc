/*
Copyright 2018, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

// RestoreFilterPlugin is a restore item action plugin for Velero
type RestoreFilterPlugin struct {
	log               logrus.FieldLogger
	configMapData     map[string]Resource
	configLoaded      bool
	configMapNotFound bool
}

// NewFilterRestorePlugin instantiates a RestorePlugin.
func NewFilterRestorePlugin(log logrus.FieldLogger) *RestoreFilterPlugin {
	log.Info("Entering Cloudcasa NewFilterRestorePlugin function.")
	defer log.Info("Exiting Cloudcasa NewFilterRestorePlugin function.")

	return &RestoreFilterPlugin{
		log:               log,
		configMapData:     make(map[string]Resource),
		configMapNotFound: false,
	}
}

// AppliesTo returns information about which resources this action should be invoked for.
// The IncludedResources and ExcludedResources slices can include both resources
// and resources with group names. These work: "ingresses", "ingresses.extensions".
// A RestoreItemAction's Execute function will only be invoked on items that match the returned
// selector. A zero-valued ResourceSelector matches all resources.
func (p *RestoreFilterPlugin) AppliesTo() (velero.ResourceSelector, error) {
	p.log.Info("Entering AppliesTo function.")
	defer p.log.Info("Exiting AppliesTo function.")

	// Log that AppliesTo is invoked but returns everything
	p.log.Info("Cloudcasa: AppliesTo will apply to all resources except namespaces.")
	return velero.ResourceSelector{
		ExcludedResources: []string{"namespaces"}, // Exclude namespaces from filtering
	}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored,
// in this case, setting a custom annotation on the item being restored.
func (p *RestoreFilterPlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {

	p.log.Info("Hello from my Cloudcasa Restore Filter Plugin!")
	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
