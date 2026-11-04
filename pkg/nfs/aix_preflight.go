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
	"strings"
)

// AIX NFS client settings that commonly break mounts which work from Linux.
// The checks only warn: an operator may have a reason for each setting, and
// the parsers below are platform-independent so they are tested on Linux.

// aixPreflightCheck is one NFS client check run at startup on AIX.
type aixPreflightCheck struct {
	name    string
	command []string
	// evaluate returns a warning, or "" if the setting looks right.
	evaluate func(output string) string
}

var aixPreflightChecks = []aixPreflightCheck{
	{
		name:    "reserved ports",
		command: []string{"/usr/sbin/nfso", "-o", "nfs_use_reserved_ports"},
		evaluate: func(out string) string {
			if v, ok := parseNfsoValue(out, "nfs_use_reserved_ports"); ok && v == "0" {
				return "nfs_use_reserved_ports is 0: NFS servers that require a privileged source port " +
					"(the Linux \"secure\" export default) will refuse mounts with an access error. " +
					"Set it with: nfso -p -o nfs_use_reserved_ports=1"
			}
			return ""
		},
	},
	{
		name:    "NFSv4 domain",
		command: []string{"/usr/sbin/chnfsdom"},
		evaluate: func(out string) string {
			if !nfsDomainIsSet(out) {
				return "no NFSv4 domain is set: on vers=4 mounts, file owners will show as nobody. " +
					"Set it to the server's domain with: chnfsdom <domain>"
			}
			return ""
		},
	},
	{
		name:    "biod",
		command: []string{"/usr/bin/lssrc", "-s", "biod"},
		evaluate: func(out string) string {
			if !srcSubsystemActive(out, "biod") {
				return "the biod subsystem is not active: NFS client I/O will be slow. Start it with: startsrc -s biod"
			}
			return ""
		},
	},
}

// parseNfsoValue reads "name = value" from `nfso -o name` output.
func parseNfsoValue(out, name string) (string, bool) {
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == name {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

// nfsDomainIsSet reads `chnfsdom` output, "Current local domain: <domain>",
// where an unset domain is shown as N/A.
func nfsDomainIsSet(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		if _, v, ok := strings.Cut(line, ":"); ok && strings.Contains(strings.ToLower(line), "domain") {
			v = strings.TrimSpace(v)
			return v != "" && !strings.EqualFold(v, "N/A")
		}
	}
	return false
}

// srcSubsystemActive reads `lssrc -s name` output:
//
//	Subsystem         Group            PID          Status
//	 biod             nfs              4194656      active
func srcSubsystemActive(out, name string) bool {
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == name && f[len(f)-1] == "active" {
			return true
		}
	}
	return false
}
