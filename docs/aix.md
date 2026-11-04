# NFS CSI driver on AIX nodes

This branch runs the NFS CSI **node plugin** on IBM AIX (7.3, `aix/ppc64`), so
pods on AIX nodes can mount NFS persistent volumes. It is meant for clusters
built from the AIX ports of the kubelet and containerd (containerd-aix, which
runs containers in WPARs).

The controller (provisioning, snapshots, deletion) is unchanged and keeps
running on Linux. An NFS share can be used from Linux and AIX pods at once
(`ReadWriteMany`), within the NFS version limits below.

## How it fits together

1. The controller Deployment on a Linux node creates the volume directory on
   the NFS server, as upstream.
2. On the AIX node the plugin runs on the host as the SRC subsystem `csi-nfs`
   (group `kubernetes`). It does not run in a pod: publishing a volume mounts
   it under the kubelet's pod directory, which from a container needs
   `Bidirectional` mount propagation. AIX has no mount propagation, and
   containerd-aix rejects it.
3. The plugin registers itself with the kubelet through
   `/var/lib/kubelet/plugins_registry`, doing node-driver-registrar's job, so
   there is one process per node instead of two.
4. On `NodePublishVolume` the plugin runs `/usr/sbin/mount` for the share at
   the pod's volume path. When the pod starts, containerd-aix grafts that path
   into the WPAR with a `namefs` mount.

The Linux node DaemonSet and Helm chart already select `kubernetes.io/os:
linux`, so they skip AIX nodes; no change is needed on the Linux side.

## What changed for AIX

- **`pkg/aixmount`**: an implementation of `k8s.io/mount-utils`' `Interface`
  for AIX. mount-utils has none (it builds a stub whose Mount, List and
  IsLikelyNotMountPoint fail). It mounts and unmounts with the AIX commands,
  reads the mount table from `mount` output, maps Linux bind mounts to
  `namefs`, translates NFS options, and kills a hung mount's whole process
  group on timeout. It has no driver dependencies so the kubelet port can
  reuse it.
- **Node server**: uses that mounter on AIX; volume stats call `statfs`
  directly (the vendored `pkg/volume/util/fs` has no AIX code); mount timeouts
  kill the AIX mount command as on Linux.
- **Kubelet registration** built into the plugin (`--kubelet-registration-path`).
  It waits until the CSI socket accepts connections before registering, and
  removes both sockets on SIGTERM (`stopsrc`) so the kubelet unregisters it.
- **Mount requests AIX cannot express** (NFSv4.1, remount) fail with
  `InvalidArgument` rather than `Internal`.
- **Startup checks** that warn about NFS client settings which commonly break
  mounts (below).
- **`deploy/aix/`**: SRC install and uninstall scripts and an AIX StorageClass.

Linux and Windows behaviour is unchanged.

## Requirements

- AIX 7.3 with `bos.net.nfs.client`.
- The AIX kubelet port with a working plugin watcher. **Not available yet**:
  fsnotify has no AIX backend, so the kubelet does not see registration
  sockets. Until the kubelet gets an AHAFS-backed or polling watcher, the
  plugin builds, starts and mounts, but the kubelet never calls it.
- containerd-aix, for the `namefs` graft into the pod's WPAR.

## Build

```sh
GOOS=aix GOARCH=ppc64 CGO_ENABLED=0 go build -o nfsplugin ./cmd/nfsplugin
```

The binary is 64-bit XCOFF and needs no AIX libraries.

## Install on a node

Copy the binary to `/opt/csi-driver-nfs/bin/nfsplugin` and run, as root:

```sh
./deploy/aix/install-aix-node.sh
```

It creates the SRC subsystem `csi-nfs`, an `/etc/inittab` entry (`csinfs`)
to start it at boot, and starts it. The node name defaults to the lower-cased hostname, which
is what the kubelet uses; set `NODE_NAME` if the kubelet is started with
`--hostname-override`. Other settings are listed at the top of the script.
Logs go to `/var/log/csi-nfs.log`.

AIX-only flags:

| Flag | Default | Purpose |
|---|---|---|
| `--kubelet-registration-path` | unset | CSI socket path to register with the kubelet. Without it the plugin does not register. |
| `--plugin-registration-dir` | `/var/lib/kubelet/plugins_registry` | the kubelet's registration directory |
| `--aix-nfs4-minor-fallback` | false | mount NFSv4.1 and v4.2 volumes as NFSv4.0 |

## Mount options

The AIX NFS client takes different options from Linux. The plugin translates
the ones StorageClasses usually carry, so one class can serve both platforms:

| Linux | AIX |
|---|---|
| `nfsvers=N`, `vers=N` (2, 3, 4, 4.0) | `vers=N` |
| `nfsvers=4.1`, `4.2`, `minorversion=1+` | refused, or `vers=4` with `--aix-nfs4-minor-fallback` |
| `nolock` | `llock` |
| `tcp`, `udp` | `proto=tcp`, `proto=udp` |
| `bg` | dropped: CSI mounts must finish in the foreground |
| `noresvport`, `nconnect`, `lookupcache`, `local_lock`, `fsc`, `*atime`, `_netdev`, `x-systemd.*` and similar | dropped (logged) |

Only tuning and bookkeeping options are dropped. Options that affect access
(`ro`, `nosuid`, `nodev`, `noexec`, `sec=`) and anything unknown are passed
through, so if AIX does not accept one the mount fails with the AIX error
rather than silently mounting with weaker settings.

**NFSv4.1 and later.** AIX supports NFS 2, 3 and 4.0; IBM has confirmed it has
no 4.1 client. The upstream example StorageClass uses `nfsvers=4.1`, so on AIX
either use a class with `nfsvers=4` or `nfsvers=3`
(`deploy/aix/storageclass-aix.yaml`), or start the plugin with
`--aix-nfs4-minor-fallback`. The fallback needs a server that still accepts
4.0, and AIX loses 4.1 features (sessions, pNFS). Some services accept 4.1+
only, such as IBM Cloud VPC file shares; AIX cannot mount those.

**Server names** are resolved by the AIX host's resolver, not cluster DNS, so
`nfs-server.default.svc.cluster.local` does not work from AIX. Use an IP
address or a name the node resolves.

## NFS client settings checked at startup

The plugin logs a warning, and carries on, when:

- `nfso -o nfs_use_reserved_ports` is 0. Linux servers export with `secure`
  by default and refuse mounts from unprivileged ports, which shows up as an
  access error. Fix: `nfso -p -o nfs_use_reserved_ports=1`.
- No NFSv4 domain is set (`chnfsdom` shows N/A). On `vers=4` mounts owners
  map to `nobody`. Fix: `chnfsdom <server's domain>`.
- The `biod` subsystem is not active. Fix: `startsrc -s biod`.

## Limitations

- No NFSv4.1 or 4.2 (see above).
- Volume stats multiply `statfs` block counts by `f_bsize`. For NFS mounts
  AIX may report the transfer size there instead of the block-count unit
  (its `statfs` also has `f_fsize`); compare with `df -k` on a real node and
  switch to `f_fsize` if they differ.
- The mount table comes from parsing `mount` output (there is no
  `/proc/mounts`, and `mntctl` is not available without cgo). Paths with
  spaces would be misread; kubelet paths have none.
- IPv6 server addresses are untested.
- Nothing here has run on AIX hardware yet. Verified so far: the
  cross-compiled build, `go vet` for aix, and unit tests on Linux, including
  the mounter run against fake `mount`/`umount` scripts that print AIX output.

## To check first on a real node

1. `mount` output for NFSv3, NFSv4 and namefs rows matches the parser
   (`pkg/aixmount/table_test.go` holds the assumed format).
2. `mount -o vers=4 host:/path dir` picks the nfs4 vfs without `-v`.
3. `umount -f` releases a mount whose server is down.
4. `chnfsdom` output format for the domain check.
5. A namefs graft of an NFS-mounted directory into a WPAR works, and
   `GetMountRefs` sees it while the pod runs.
