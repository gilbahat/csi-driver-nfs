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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

// kubeletRegistration does what node-driver-registrar does, inside the
// plugin: it serves the kubelet's plugin registration API on a socket in the
// kubelet's registration directory, which the kubelet's plugin watcher finds
// and calls back on.
//
// On AIX the node plugin runs on the host as an SRC subsystem, not in a pod,
// so there is no sidecar to do this. Folding it in keeps one process per node
// instead of two subsystems that must be kept in step.
type kubeletRegistration struct {
	registerapi.UnimplementedRegistrationServer

	driverName string
	// endpoint is the CSI socket path the kubelet should dial.
	endpoint string
	// onFailure is called when the kubelet reports that registration failed.
	onFailure func(msg string)
}

// registrationSocketPath is where node-driver-registrar would put the socket.
func registrationSocketPath(dir, driverName string) string {
	return filepath.Join(dir, driverName+"-reg.sock")
}

// GetInfo tells the kubelet which CSI driver this is and where it listens.
func (r *kubeletRegistration) GetInfo(_ context.Context, _ *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	klog.V(2).Infof("kubelet registration: GetInfo for %s", r.driverName)
	return &registerapi.PluginInfo{
		Type:              registerapi.CSIPlugin,
		Name:              r.driverName,
		Endpoint:          r.endpoint,
		SupportedVersions: []string{"1.0.0"},
	}, nil
}

// NotifyRegistrationStatus receives the kubelet's verdict. A failure is
// fatal, as in node-driver-registrar, so the service manager restarts the
// plugin and registration is tried again.
func (r *kubeletRegistration) NotifyRegistrationStatus(_ context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	if !status.PluginRegistered {
		msg := fmt.Sprintf("kubelet failed to register %s: %s", r.driverName, status.Error)
		klog.Error(msg)
		if r.onFailure != nil {
			r.onFailure(msg)
		}
		return &registerapi.RegistrationStatusResponse{}, nil
	}
	klog.Infof("kubelet registered %s (endpoint %s)", r.driverName, r.endpoint)
	return &registerapi.RegistrationStatusResponse{}, nil
}

// serveKubeletRegistration starts the registration server and returns a
// function that stops it.
func serveKubeletRegistration(dir string, reg *kubeletRegistration) (func(), error) {
	if reg.driverName == "" || reg.endpoint == "" {
		return nil, fmt.Errorf("driver name and endpoint are required for kubelet registration")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("creating registration directory: %w", err)
	}
	sock := registrationSocketPath(dir, reg.driverName)
	if err := os.Remove(sock); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("removing stale registration socket %s: %w", sock, err)
	}
	lis, err := net.Listen("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", sock, err)
	}
	server := grpc.NewServer()
	registerapi.RegisterRegistrationServer(server, reg)
	go func() {
		klog.Infof("kubelet registration: serving %s", sock)
		if err := server.Serve(lis); err != nil {
			klog.Errorf("kubelet registration server stopped: %v", err)
		}
	}()
	return func() {
		server.Stop()
		_ = os.Remove(sock)
	}, nil
}

// waitForSocket waits until something accepts connections on the unix
// socket at path, so the kubelet is not told about a CSI endpoint that is not
// serving yet: a failed NodeGetInfo during registration is fatal. Checking
// that the file exists is not enough, since a socket left by a previous run
// stays on disk until the gRPC server replaces it.
func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("unix", path, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("CSI socket %s is not accepting connections after %v: %w", path, timeout, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
