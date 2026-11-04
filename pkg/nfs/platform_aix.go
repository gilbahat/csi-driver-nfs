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
	"context"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"

	"github.com/kubernetes-csi/csi-driver-nfs/pkg/aixmount"
)

const (
	defaultAIXPluginRegistrationDir = "/var/lib/kubelet/plugins_registry"
	csiSocketWaitTimeout            = 30 * time.Second
	preflightCommandTimeout         = 10 * time.Second
)

// newNodeMounter returns the AIX mounter: k8s.io/mount-utils has no AIX
// implementation.
func newNodeMounter(n *Driver) mount.Interface {
	return aixmount.New(aixmount.NFSOptionPolicy{NFS4MinorFallback: n.platform.NFS4MinorFallback})
}

// startPlatformServices runs the AIX NFS client checks and, if configured,
// registers the plugin with the kubelet.
func startPlatformServices(n *Driver) {
	runAIXPreflight()

	if n.platform.KubeletRegistrationPath == "" {
		klog.Warning("--kubelet-registration-path is not set: the plugin will not register with the kubelet")
		return
	}
	dir := n.platform.PluginRegistrationDir
	if dir == "" {
		dir = defaultAIXPluginRegistrationDir
	}
	if proto, addr, err := ParseEndpoint(n.endpoint); err == nil && proto == "unix" {
		if err := waitForSocket("/"+addr, csiSocketWaitTimeout); err != nil {
			klog.Fatalf("not registering with the kubelet: %v", err)
		}
	}
	reg := &kubeletRegistration{
		driverName: n.name,
		endpoint:   n.platform.KubeletRegistrationPath,
		onFailure: func(string) {
			// Let SRC restart the subsystem, as the kubelet would restart a
			// failed node-driver-registrar container.
			os.Exit(1)
		},
	}
	stop, err := serveKubeletRegistration(dir, reg)
	if err != nil {
		klog.Fatalf("kubelet registration: %v", err)
	}
	go cleanUpOnSignal(n, stop)
}

// cleanUpOnSignal removes the registration and CSI sockets when SRC stops
// the subsystem (stopsrc sends SIGTERM). The kubelet unregisters a plugin
// only when its registration socket disappears, so a socket left behind would
// keep a stopped plugin looking registered.
func cleanUpOnSignal(n *Driver, stopRegistration func()) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	<-ctx.Done()
	klog.Info("stopping: removing kubelet registration and CSI sockets")
	stopRegistration()
	if proto, addr, err := ParseEndpoint(n.endpoint); err == nil && proto == "unix" {
		_ = os.Remove("/" + addr)
	}
	klog.Flush()
	os.Exit(0)
}

// runAIXPreflight logs warnings for NFS client settings that commonly make
// mounts fail or misbehave. It never stops the plugin.
func runAIXPreflight() {
	for _, c := range aixPreflightChecks {
		ctx, cancel := context.WithTimeout(context.Background(), preflightCommandTimeout)
		out, err := exec.CommandContext(ctx, c.command[0], c.command[1:]...).CombinedOutput()
		cancel()
		if err != nil {
			klog.Warningf("AIX preflight %q: %s: %v", c.name, strings.Join(c.command, " "), err)
			continue
		}
		if w := c.evaluate(string(out)); w != "" {
			klog.Warningf("AIX preflight %q: %s", c.name, w)
		} else {
			klog.V(2).Infof("AIX preflight %q: ok", c.name)
		}
	}
}
