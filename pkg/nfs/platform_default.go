//go:build !aix

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
	"runtime"

	mount "k8s.io/mount-utils"
)

// newNodeMounter returns the mounter the node server uses.
func newNodeMounter(_ *Driver) mount.Interface {
	mounter := mount.New("")
	if runtime.GOOS == "linux" {
		// MounterForceUnmounter is only implemented on Linux now
		mounter = mounter.(mount.MounterForceUnmounter)
	}
	return mounter
}

// startPlatformServices starts anything the platform needs beside the CSI
// gRPC server. Nothing on Linux and Windows: node-driver-registrar runs as a
// sidecar there.
func startPlatformServices(_ *Driver) {}
