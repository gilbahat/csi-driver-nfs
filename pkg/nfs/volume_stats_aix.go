//go:build aix

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
	"fmt"

	"golang.org/x/sys/unix"
	"k8s.io/kubernetes/pkg/volume"
)

// getVolumeMetrics returns usage for a mounted volume. The vendored
// k8s.io/kubernetes/pkg/volume/util/fs has no AIX implementation, so statfs
// is called directly. AIX's statfs has no f_frsize; block counts are in
// f_bsize units.
func getVolumeMetrics(path string) (*volume.Metrics, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return nil, fmt.Errorf("statfs %s: %w", path, err)
	}
	return metricsFromStatfs(int64(st.Bsize), int64(st.Blocks), int64(st.Bfree), int64(st.Bavail),
		int64(st.Files), int64(st.Ffree)), nil
}
