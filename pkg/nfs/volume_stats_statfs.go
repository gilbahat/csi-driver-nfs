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

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/volume"
)

// metricsFromStatfs computes volume metrics from statfs fields the same way
// k8s.io/kubernetes/pkg/volume/util/fs does on Linux. It is used on
// platforms that package does not cover (AIX).
func metricsFromStatfs(bsize, blocks, bfree, bavail, files, ffree int64) *volume.Metrics {
	return &volume.Metrics{
		Available:  resource.NewQuantity(bavail*bsize, resource.BinarySI),
		Capacity:   resource.NewQuantity(blocks*bsize, resource.BinarySI),
		Used:       resource.NewQuantity((blocks-bfree)*bsize, resource.BinarySI),
		Inodes:     resource.NewQuantity(files, resource.BinarySI),
		InodesFree: resource.NewQuantity(ffree, resource.BinarySI),
		InodesUsed: resource.NewQuantity(files-ffree, resource.BinarySI),
	}
}
