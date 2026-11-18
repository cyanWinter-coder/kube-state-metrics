/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package metric

import (
	"bytes"
	"strings"
	"testing"
)

func TestFamilyString(t *testing.T) {
	m := Metric{
		LabelKeys:   []string{"namespace"},
		LabelValues: []string{"default"},
		Value:       1,
	}

	f := Family{
		Name:    "kube_pod_info",
		Metrics: []*Metric{&m},
	}

	expected := "kube_pod_info{namespace=\"default\"} 1"
	got := strings.TrimSpace(string(f.ByteSlice()))

	if got != expected {
		t.Fatalf("expected %v but got %v", expected, got)
	}
}

// The metrics store deduplicates series on the rendered text: it assumes a
// family renders as one newline-terminated line per metric, each starting
// with the family name, with the value separated from the labels by the last
// space on the line. This test pins that contract.
func TestFamilyRendersOneSeriesPerLine(t *testing.T) {
	f := Family{
		Name: "kube_pod_status_ready",
		Metrics: []*Metric{
			{
				LabelKeys:   []string{"namespace", "pod", "condition"},
				LabelValues: []string{"default", "pod-a", "true"},
				Value:       1,
			},
			{
				LabelKeys:   []string{"namespace", "pod", "condition"},
				LabelValues: []string{"default", "pod-a", "false"},
				Value:       0,
			},
		},
	}

	got := string(f.ByteSlice())
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected every series line to be newline-terminated, got %q", got)
	}
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected one line per metric, got %d lines in %q", len(lines), got)
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "kube_pod_status_ready{") {
			t.Fatalf("expected each series line to start with the family name, got %q", line)
		}
	}
}

func BenchmarkMetricWrite(b *testing.B) {
	tests := []struct {
		testName       string
		metric         Metric
		expectedLength int
	}{
		{
			testName: "value-1",
			metric: Metric{
				LabelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
				LabelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
				Value:       float64(1),
			},
			expectedLength: 145,
		},
		{
			testName: "value-35.7",
			metric: Metric{
				LabelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
				LabelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
				Value:       35.7,
			},
			expectedLength: 148,
		},
	}

	for _, test := range tests {
		b.Run(test.testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := bytes.Buffer{}

				test.metric.Write(&builder)

				s := builder.Bytes()

				// Ensuring that the string is actually build, not optimized
				// away by compilation.
				got := len(s)
				if test.expectedLength != got {
					b.Fatalf("expected string of length %v but got %v", test.expectedLength, got)
				}
			}
		})
	}
}
