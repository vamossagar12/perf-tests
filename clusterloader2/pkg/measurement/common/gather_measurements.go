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

type gatherMetricsMeasurement struct {}

func (g gatherMetricsMeasurement) Execute(config *measurement.MeasurementConfig) ([]measurement.Summary, error) {
	identifiersFromConfig, err := util.GetString(config.Params, "Identifiers")
	if err != nil {
		return nil, err
	}
	identifiers := strings.Split(strings.TrimSpace(identifiersFromConfig), ",")
	methodName, err := util.GetString(config.Params, "Method")
	if err != nil {
		return nil, err
	}

	var wg wait.Group
	errList := errors.NewErrorList()

	for i := range identifiers {
		index := i
		measurementInstance, err := config.MeasurementManager.GetMeasurementInstance(methodName, identifiers[index])
		if err != nil{
			errList.Append(fmt.Errorf("could not fetch measurement using identifier %s - method %s error: %v", identifiers[index], methodName, err))
			continue
		}
		wg.Start(func() {
			_, err := measurementInstance.Execute(config)
			if err != nil {
				errList.Append(fmt.Errorf("measurement call %s - %s error: %v", methodName, identifiers[index], err))
			}
		})
	}
	return nil, errList
}

func (g gatherMetricsMeasurement) Dispose() {}

func (g gatherMetricsMeasurement) String() string {
	return gatherMeasurementsMetricName
}
