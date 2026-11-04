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

# Installs the NFS CSI node plugin on an AIX node as an SRC subsystem.
#
# The node plugin cannot run as a pod on AIX: mounting into the kubelet's pod
# directories needs Bidirectional mount propagation, which AIX does not have
# and containerd-aix refuses. It runs on the host instead, as the kubelet and
# amco-agent do, and registers itself with the kubelet (no
# node-driver-registrar sidecar).
#
# Settings, all optional, from the environment:
#   BIN          plugin binary            (/opt/csi-driver-nfs/bin/nfsplugin)
#   NODE_NAME    Kubernetes node name     (lower-cased hostname, as the kubelet uses)
#   DRIVER_NAME  CSI driver name          (nfs.csi.k8s.io)
#   KUBELET_DIR  kubelet root directory   (/var/lib/kubelet)
#   SUBSYS       SRC subsystem name       (csi-nfs)
#   GROUP        SRC group                (kubernetes)
#   LOG          log file                 (/var/log/csi-nfs.log)
#   EXTRA_ARGS   more plugin flags, e.g. "--aix-nfs4-minor-fallback"
#   NO_INITTAB   set to 1 to skip the /etc/inittab entry that starts it at boot

set -e

BIN=${BIN:-/opt/csi-driver-nfs/bin/nfsplugin}
NODE_NAME=${NODE_NAME:-$(hostname | tr '[:upper:]' '[:lower:]')}
DRIVER_NAME=${DRIVER_NAME:-nfs.csi.k8s.io}
KUBELET_DIR=${KUBELET_DIR:-/var/lib/kubelet}
SUBSYS=${SUBSYS:-csi-nfs}
GROUP=${GROUP:-kubernetes}
LOG=${LOG:-/var/log/csi-nfs.log}
EXTRA_ARGS=${EXTRA_ARGS:-}

SOCK_DIR=${KUBELET_DIR}/plugins/csi-nfsplugin
REG_DIR=${KUBELET_DIR}/plugins_registry
SOCK=${SOCK_DIR}/csi.sock

if [ "$(uname -s)" != "AIX" ]; then
	echo "this script is for AIX" >&2
	exit 1
fi
if [ ! -x "${BIN}" ]; then
	echo "${BIN} not found or not executable" >&2
	exit 1
fi
if ! lslpp -l bos.net.nfs.client >/dev/null 2>&1; then
	echo "bos.net.nfs.client is not installed; NFS mounts will fail" >&2
	exit 1
fi

mkdir -p "${SOCK_DIR}" "${REG_DIR}"
chmod 0750 "${SOCK_DIR}"

ARGS="--v=2 --nodeid=${NODE_NAME} --drivername=${DRIVER_NAME} --endpoint=unix://${SOCK}"
ARGS="${ARGS} --kubelet-registration-path=${SOCK} --plugin-registration-dir=${REG_DIR}"
ARGS="${ARGS} ${EXTRA_ARGS}"

if lssrc -s "${SUBSYS}" >/dev/null 2>&1; then
	stopsrc -s "${SUBSYS}" >/dev/null 2>&1 || true
	rmssys -s "${SUBSYS}"
fi

# -R: restart on abnormal exit (a failed kubelet registration exits 1)
# -Q: one instance; -S -n 15 -f 9: stop with SIGTERM, force with SIGKILL
mkssys -s "${SUBSYS}" -p "${BIN}" -a "${ARGS}" -u 0 -G "${GROUP}" \
	-R -Q -S -n 15 -f 9 -w 30 -o "${LOG}" -e "${LOG}"

# inittab identifiers are at most 14 characters; derive it from SUBSYS so a
# reinstall under another name does not leave the old entry starting the old
# subsystem.
ITAB_ID=$(echo "${SUBSYS}" | tr -cd '[:alnum:]' | cut -c1-14)
if [ "${NO_INITTAB}" != "1" ]; then
	rmitab "${ITAB_ID}" >/dev/null 2>&1 || true
	mkitab "${ITAB_ID}:2:once:/usr/bin/startsrc -s ${SUBSYS} >/dev/console 2>&1"
fi

startsrc -s "${SUBSYS}"
echo "${SUBSYS} installed for node ${NODE_NAME}; logs in ${LOG}"
