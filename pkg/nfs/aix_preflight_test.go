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

import "testing"

func preflightCheck(t *testing.T, name string) aixPreflightCheck {
	t.Helper()
	for _, c := range aixPreflightChecks {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("no preflight check %q", name)
	return aixPreflightCheck{}
}

func TestAIXPreflightReservedPorts(t *testing.T) {
	c := preflightCheck(t, "reserved ports")
	if w := c.evaluate("nfs_use_reserved_ports = 0\n"); w == "" {
		t.Error("expected a warning for nfs_use_reserved_ports = 0")
	}
	if w := c.evaluate("nfs_use_reserved_ports = 1\n"); w != "" {
		t.Errorf("unexpected warning: %s", w)
	}
	if w := c.evaluate("garbage"); w != "" {
		t.Errorf("unparseable output should not warn: %s", w)
	}
}

func TestAIXPreflightNFSDomain(t *testing.T) {
	c := preflightCheck(t, "NFSv4 domain")
	if w := c.evaluate("Current local domain: N/A\n"); w == "" {
		t.Error("expected a warning for an unset domain")
	}
	if w := c.evaluate("Current local domain: example.com\n"); w != "" {
		t.Errorf("unexpected warning: %s", w)
	}
}

func TestAIXPreflightBiod(t *testing.T) {
	c := preflightCheck(t, "biod")
	active := "Subsystem         Group            PID          Status \n biod             nfs              4194656      active\n"
	if w := c.evaluate(active); w != "" {
		t.Errorf("unexpected warning: %s", w)
	}
	inoperative := "Subsystem         Group            PID          Status \n biod             nfs                           inoperative\n"
	if w := c.evaluate(inoperative); w == "" {
		t.Error("expected a warning for inoperative biod")
	}
}
