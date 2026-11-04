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

// PlatformOptions holds driver options that only some node platforms use.
// On Linux and Windows they are ignored.
type PlatformOptions struct {
	// NFS4MinorFallback (AIX) mounts NFSv4.1 and v4.2 volumes as NFSv4.0,
	// the only NFSv4 version the AIX client speaks. Off by default, so such
	// volumes fail to mount with an explanatory error instead.
	NFS4MinorFallback bool

	// KubeletRegistrationPath (AIX) is the CSI socket path to register with
	// the kubelet. When set, the plugin registers itself through the
	// kubelet's plugin registration directory, doing the job of
	// node-driver-registrar, which cannot run as a pod on AIX.
	KubeletRegistrationPath string

	// PluginRegistrationDir (AIX) is the kubelet's plugin registration
	// directory, normally /var/lib/kubelet/plugins_registry.
	PluginRegistrationDir string
}
