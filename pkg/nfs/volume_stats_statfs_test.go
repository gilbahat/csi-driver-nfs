/*
Copyright 2026 The Kubernetes Authors.

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

package nfs

import "testing"

func TestMetricsFromStatfs(t *testing.T) {
	m := metricsFromStatfs(4096, 1000, 400, 300, 50, 20)
	check := func(name string, got, want int64) {
		if got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
	check("Capacity", m.Capacity.Value(), 4096000)
	check("Available", m.Available.Value(), 1228800)
	check("Used", m.Used.Value(), 2457600)
	check("Inodes", m.Inodes.Value(), 50)
	check("InodesFree", m.InodesFree.Value(), 20)
	check("InodesUsed", m.InodesUsed.Value(), 30)
}
