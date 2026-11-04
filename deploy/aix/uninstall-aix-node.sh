#!/usr/bin/ksh
# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Removes the NFS CSI node plugin SRC subsystem from an AIX node. Drain the
# node first: NFS volumes still mounted for pods stay mounted, and the
# kubelet cannot unmount them once the plugin is gone.

SUBSYS=${SUBSYS:-csi-nfs}
DRIVER_NAME=${DRIVER_NAME:-nfs.csi.k8s.io}
KUBELET_DIR=${KUBELET_DIR:-/var/lib/kubelet}

stopsrc -s "${SUBSYS}" >/dev/null 2>&1 || true
rmssys -s "${SUBSYS}" >/dev/null 2>&1 || true
rmitab "$(echo "${SUBSYS}" | tr -cd '[:alnum:]' | cut -c1-14)" >/dev/null 2>&1 || true
rm -f "${KUBELET_DIR}/plugins_registry/${DRIVER_NAME}-reg.sock" \
	"${KUBELET_DIR}/plugins/csi-nfsplugin/csi.sock"
echo "${SUBSYS} removed"
