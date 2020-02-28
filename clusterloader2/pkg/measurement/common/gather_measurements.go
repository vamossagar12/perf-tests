/*
Copyright 2018 The Kubernetes Authors.

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

package common

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/perf-tests/clusterloader2/pkg/errors"
	"k8s.io/perf-tests/clusterloader2/pkg/measurement"
	"k8s.io/perf-tests/clusterloader2/pkg/util"
	"strings"
)

const (
	gatherMeasurementsMetricName = "GatherMeasurements"
)

func init() {
	if err := measurement.Register(gatherMeasurementsMetricName, createGatherMeasurementsMeasurement); err != nil {
		klog.Fatalf("Cannot register %s: %v", gatherMeasurementsMetricName, err)
	}
}

func createGatherMeasurementsMeasurement() measurement.Measurement {
	return &gatherMetricsMeasurement{}
}

type gatherMetricsMeasurement struct{}

func (g gatherMetricsMeasurement) Execute(config *measurement.MeasurementConfig) ([]measurement.Summary, error) {

	// Assuming that Method config would be present only once and won't/shouldn't be overridden
	errList := errors.NewErrorList()
	methodName, err := util.GetString(config.Params, "Method")
	if err != nil {
		return nil, err
	}

	identifiersWithConfig, err := util.GetMap(config.Params, "Identifiers")
	if err != nil {
		return nil, err
	}

	var wg wait.Group

	for identifier:= range identifiersWithConfig {
		measurementInstance, err := config.MeasurementManager.GetMeasurementInstance(methodName, identifier)

		if err != nil {
			errList.Append(fmt.Errorf("could not fetch measurement using identifier %s - method %s error: %v", identifier, methodName, err))
			continue
		}
		// A new params map is being created since within a measurement which we try to wrap,
		// there might still be cases where the individual params may be measurement specific
		measurementParams := map[string]interface{}{}

		// We will first add the identifier specific configs if there are defined
		if identifiersWithConfig[identifier] != nil {
			for configKey, configVal:= range identifiersWithConfig[identifier].(map[string]interface{}){
				measurementParams[configKey] = strings.TrimSpace(configVal.(string))
			}
		}

		// These are the common configs defined outside the Identifiers block, which apply to all the Identifiers.
		// Note that we are processing the common config multiple tine for each identifier which is duplicated work
		// but since it's a small set of configs so for brevity's sake, keeping it this way for now.
		for k := range config.Params{
			if k == "Identifiers" || k == "Method" {
				continue
			}
			configParam, err := util.GetString(config.Params, k)
			if err != nil {
				errList.Append(fmt.Errorf("error fetching param for identifier %s - method %s - key %s error: %v", identifier, methodName, k, err))
				continue
			}
			measurementParams[k] = strings.TrimSpace(configParam)
		}

		wrappedMeasurementConfig := measurement.MeasurementConfig{
			ClusterFramework:    config.ClusterFramework,
			PrometheusFramework: config.PrometheusFramework,
			Params:              measurementParams,
			TemplateProvider:    config.TemplateProvider,
			Identifier:          identifier,
			CloudProvider:       config.ClusterLoaderConfig.ClusterConfig.Provider,
			ClusterLoaderConfig: config.ClusterLoaderConfig,
			MeasurementManager:  config.MeasurementManager,
		}

		wg.Start(func() {
			_, err := measurementInstance.Execute(&wrappedMeasurementConfig)
			if err != nil {
				errList.Append(fmt.Errorf("measurement call %s - %s error: %v", methodName, identifier, err))
				klog.Infof("Errors size: %d", len(errList.Error()))
			}
		})
	}
	return nil, errList
}

func (g gatherMetricsMeasurement) Dispose() {}

func (g gatherMetricsMeasurement) String() string {
	return gatherMeasurementsMetricName
}
