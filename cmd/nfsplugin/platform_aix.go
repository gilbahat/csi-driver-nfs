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

package main

import (
	"flag"

	"github.com/kubernetes-csi/csi-driver-nfs/pkg/nfs"
)

// Registered only on AIX, so Linux and Windows images keep their flag set.
var (
	kubeletRegistrationPath = flag.String("kubelet-registration-path", "", "CSI socket path to register with the kubelet; the plugin registers itself only when this is set (it runs on the host, where node-driver-registrar cannot)")
	pluginRegistrationDir   = flag.String("plugin-registration-dir", "/var/lib/kubelet/plugins_registry", "the kubelet's plugin registration directory")
	aixNFS4MinorFallback    = flag.Bool("aix-nfs4-minor-fallback", false, "mount NFSv4.1 and v4.2 volumes as NFSv4.0, the only NFSv4 version the AIX client speaks")
)

func platformOptions() nfs.PlatformOptions {
	return nfs.PlatformOptions{
		NFS4MinorFallback:       *aixNFS4MinorFallback,
		KubeletRegistrationPath: *kubeletRegistrationPath,
		PluginRegistrationDir:   *pluginRegistrationDir,
	}
}
