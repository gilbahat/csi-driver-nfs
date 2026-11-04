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
	"errors"
	"fmt"
	"time"

	mount "k8s.io/mount-utils"

	"github.com/kubernetes-csi/csi-driver-nfs/pkg/aixmount"
)

// mountNFSWithTimeout mounts with the AIX mounter, killing the mount command
// and its children when the timeout expires, as the Linux version does. An
// AIX foreground NFS mount keeps retrying an unreachable server for a long
// time, so a mount that is merely abandoned would linger.
func mountNFSWithTimeout(mounter mount.Interface, source, targetPath string, mountOptions []string, timeout time.Duration) error {
	timeoutErr := fmt.Errorf("mount volume %s to %s timeout after %ds", source, targetPath, int(timeout/time.Second))

	m, ok := mounter.(*aixmount.Mounter)
	if !ok {
		return WaitUntilTimeout(timeout,
			func() error { return mounter.Mount(source, targetPath, "nfs", mountOptions) },
			func() error { return timeoutErr })
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := m.MountContext(ctx, source, targetPath, "nfs", mountOptions, nil); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return timeoutErr
		}
		return err
	}
	return nil
}
