/*
Copyright 2019 The Knative Authors

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

package stats

import (
	"errors"
	"testing"
	"time"

	"go.opencensus.io/stats"

	"knative.dev/pkg/metrics/metricskey"
	"knative.dev/pkg/metrics/metricstest"
)

const (
	testNs      = "test-default"
	testSvc     = "helloworld-go-service"
	testConf    = "helloworld-go"
	testRev     = "helloworld-go-00001"
	testPod     = "helloworld-go-00001-abcd"
	countName   = "request_count"
	qdepthName  = "queue_depth"
	latencyName = "request_latencies"
)

var (
	queueSizeMetric = stats.Int64(
		qdepthName,
		"Queue size",
		stats.UnitDimensionless)
	countMetric = stats.Int64(
		countName,
		"The number of requests that are routed to queue-proxy",
		stats.UnitDimensionless)
	latencyMetric = stats.Float64(
		latencyName,
		"The response time in millisecond",
		stats.UnitMilliseconds)
)

func TestNewStatsReporterNegative(t *testing.T) {
	tests := []struct {
		name      string
		errorMsg  string
		result    error
		namespace string
		config    string
		revision  string
	}{{
		"Empty_Namespace_Value",
		"Expected namespace empty error",
		errors.New("namespace must not be empty"),
		"",
		testConf,
		testRev,
	}, {
		"Empty_Config_Value",
		"Expected config empty error",
		errors.New("config must not be empty"),
		testNs,
		"",
		testRev,
	}, {
		"Empty_Revision_Value",
		"Expected revision empty error",
		errors.New("revision must not be empty"),
		testRev,
		testConf,
		"",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewStatsReporter(test.namespace, testSvc, test.config, test.revision, testPod,
				countMetric, latencyMetric, queueSizeMetric); err.Error() != test.result.Error() {
				t.Errorf("%+v, got: '%+v'", test.errorMsg, err)
			}
		})
	}
}

func TestReporterReport(t *testing.T) {
	r, err := NewStatsReporter(testNs, testSvc, testConf, testRev, testPod, countMetric, latencyMetric, queueSizeMetric)
	if err != nil {
		t.Fatalf("Unexpected error from NewStatsReporter() = %v", err)
	}
	wantTags := map[string]string{
		metricskey.LabelNamespaceName:     testNs,
		metricskey.LabelServiceName:       testSvc,
		metricskey.LabelConfigurationName: testConf,
		metricskey.LabelRevisionName:      testRev,
		"pod_name":                        testPod,
		"container_name":                  "queue-proxy",
		"response_code":                   "200",
		"response_code_class":             "2xx",
	}

	// Send statistics only once and observe the results
	r.ReportRequestCount(200)
	metricstest.CheckCountData(t, "request_count", wantTags, 1)

	// The stats are cumulative - record multiple entries, should get sum
	r.ReportRequestCount(200)
	r.ReportRequestCount(200)
	metricstest.CheckCountData(t, "request_count", wantTags, 3)

	// Send statistics only once and observe the results
	r.ReportResponseTime(200, 100*time.Millisecond)
	metricstest.CheckDistributionData(t, "request_latencies", wantTags, 1, 100, 100)

	// The stats are cumulative - record multiple entries, should get count sum
	r.ReportResponseTime(200, 200*time.Millisecond)
	r.ReportResponseTime(200, 300*time.Millisecond)
	metricstest.CheckDistributionData(t, "request_latencies", wantTags, 3, 100, 300)

	wantTags = map[string]string{
		metricskey.LabelNamespaceName:     testNs,
		metricskey.LabelServiceName:       testSvc,
		metricskey.LabelConfigurationName: testConf,
		metricskey.LabelRevisionName:      testRev,
		"pod_name":                        testPod,
		"container_name":                  "queue-proxy",
	}
	r.ReportQueueDepth(1)
	r.ReportQueueDepth(2)
	metricstest.CheckLastValueData(t, "queue_depth", wantTags, 2)

	metricstest.Unregister(countName, latencyName, qdepthName)

	// Test reporter with empty service name
	r, err = NewStatsReporter(testNs, "" /*service name*/, testConf, "testrev_svc_missing", testPod, countMetric, latencyMetric, queueSizeMetric)
	if err != nil {
		t.Fatalf("Unexpected error from NewStatsReporter() = %v", err)
	}
	wantTags = map[string]string{
		metricskey.LabelNamespaceName:     testNs,
		metricskey.LabelServiceName:       "unknown",
		metricskey.LabelConfigurationName: testConf,
		metricskey.LabelRevisionName:      "testrev_svc_missing",
		"pod_name":                        testPod,
		"container_name":                  "queue-proxy",
		"response_code":                   "200",
		"response_code_class":             "2xx",
	}

	// Send statistics only once and observe the results
	r.ReportRequestCount(200)
	metricstest.CheckCountData(t, "request_count", wantTags, 1)

	metricstest.Unregister(countName, latencyName, qdepthName)
}
