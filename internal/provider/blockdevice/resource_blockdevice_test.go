package blockdevice

import (
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newDevice(spec map[string]interface{}, labels map[string]interface{}) *unstructured.Unstructured {
	metadata := map[string]interface{}{"name": "bd", "namespace": "longhorn-system"}
	if labels != nil {
		metadata["labels"] = labels
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "harvesterhci.io/v1beta1",
		"kind":       "BlockDevice",
		"metadata":   metadata,
		"spec":       spec,
	}}
}

func resourceData(t *testing.T, raw map[string]interface{}) *schema.ResourceData {
	t.Helper()
	raw["name"] = "bd"
	raw["namespace"] = "longhorn-system"
	return schema.TestResourceDataRaw(t, Schema(), raw)
}

func TestApplyBlockDeviceSpecDefaultsToLonghornV1(t *testing.T) {
	d := resourceData(t, map[string]interface{}{"provision": true, "force_formatted": true})
	obj := newDevice(map[string]interface{}{"devPath": "/dev/vdb", "nodeName": "n1"}, nil)

	applyBlockDeviceSpec(d, obj)

	if engine, _, _ := unstructured.NestedString(obj.Object, "spec", "provisioner", "longhorn", "engineVersion"); engine != engineLonghornV1 {
		t.Errorf("engineVersion = %q, want %q", engine, engineLonghornV1)
	}
	if _, found, _ := unstructured.NestedString(obj.Object, "spec", "provisioner", "longhorn", "diskDriver"); found {
		t.Error("diskDriver must not be set for LonghornV1")
	}
	if ff, _, _ := unstructured.NestedBool(obj.Object, "spec", "fileSystem", "forceFormatted"); !ff {
		t.Error("forceFormatted = false, want true")
	}
}

func TestApplyBlockDeviceSpecKeepsExistingProvisioner(t *testing.T) {
	d := resourceData(t, map[string]interface{}{"provision": false})
	obj := newDevice(map[string]interface{}{
		"provision":   true,
		"provisioner": map[string]interface{}{"longhorn": map[string]interface{}{"engineVersion": "LonghornV1"}},
	}, nil)

	applyBlockDeviceSpec(d, obj)

	if _, found, _ := unstructured.NestedMap(obj.Object, "spec", "provisioner"); !found {
		t.Error("the provisioner of an existing device must be kept")
	}
	if p, _, _ := unstructured.NestedBool(obj.Object, "spec", "provision"); p {
		t.Error("provision = true, want false")
	}
}

func TestApplyProvisionerDiskDriverOnlyForV2(t *testing.T) {
	cases := []struct {
		name, engine, driver, wantEngine, wantDriver string
	}{
		{"V1 ignores the driver", "LonghornV1", "auto", "LonghornV1", ""},
		{"V2 keeps the driver", "LonghornV2", "aio", "LonghornV2", "aio"},
		{"empty engine means V1", "", "", "LonghornV1", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := newDevice(map[string]interface{}{}, nil)
			applyProvisioner(map[string]interface{}{
				"longhorn": []interface{}{map[string]interface{}{"engine_version": tc.engine, "disk_driver": tc.driver}},
			}, obj)
			engine, _, _ := unstructured.NestedString(obj.Object, "spec", "provisioner", "longhorn", "engineVersion")
			driver, _, _ := unstructured.NestedString(obj.Object, "spec", "provisioner", "longhorn", "diskDriver")
			if engine != tc.wantEngine || driver != tc.wantDriver {
				t.Errorf("got engine %q driver %q, want %q %q", engine, driver, tc.wantEngine, tc.wantDriver)
			}
		})
	}
}

func TestApplyBlockDeviceSpecTagsAndLabels(t *testing.T) {
	d := resourceData(t, map[string]interface{}{
		"tags":   map[string]interface{}{"role": "data"},
		"labels": map[string]interface{}{"tier": "fast"},
	})
	obj := newDevice(map[string]interface{}{}, map[string]interface{}{
		"kubernetes.io/hostname":          "n1",
		"ndm.harvesterhci.io/device-type": "disk",
		"tags.harvesterhci.io/legacy":     "x",
		"tag.harvesterhci.io/stale":       "x",
		"stale-user-label":                "x",
	})

	applyBlockDeviceSpec(d, obj)

	got := obj.GetLabels()
	want := map[string]string{
		"kubernetes.io/hostname":          "n1",
		"ndm.harvesterhci.io/device-type": "disk",
		"tag.harvesterhci.io/role":        "data",
		"tier":                            "fast",
	}
	if len(got) != len(want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("label %s = %q, want %q (all labels: %v)", k, got[k], v, got)
		}
	}
}

func TestUnprovisionSpecOnlyFlipsProvision(t *testing.T) {
	obj := newDevice(map[string]interface{}{
		"provision":   true,
		"tags":        []interface{}{"ssd"},
		"provisioner": map[string]interface{}{"longhorn": map[string]interface{}{"engineVersion": "LonghornV1"}},
	}, nil)

	if !unprovisionSpec(obj) {
		t.Fatal("unprovisionSpec() = false for a provisioned device")
	}
	if p, _, _ := unstructured.NestedBool(obj.Object, "spec", "provision"); p {
		t.Error("provision still true")
	}
	if _, found, _ := unstructured.NestedMap(obj.Object, "spec", "provisioner"); !found {
		t.Error("provisioner removed, the node disk manager webhook would restore the previous spec")
	}
	if tags, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "tags"); len(tags) != 1 {
		t.Errorf("device tags changed: %v", tags)
	}
	if unprovisionSpec(obj) {
		t.Error("unprovisionSpec() = true for a device that is already unprovisioned")
	}
}

func TestBlockDeviceProgress(t *testing.T) {
	since := time.Date(2026, 9, 24, 16, 21, 0, 0, time.UTC)
	condition := func(status, reason, at string) map[string]interface{} {
		return map[string]interface{}{
			"type": "Mounted", "status": status, "reason": reason, "lastUpdateTime": at,
			"message": "failed to update device mount: wrong fs type, bad option, bad superblock on /dev/vdb\n",
		}
	}
	cases := []struct {
		name        string
		phase       string
		conditions  []interface{}
		wantFailure bool
	}{
		{"provisioned", "Provisioned", nil, false},
		{"mount error after the change", "Unprovisioned", []interface{}{condition("False", "Error", "2026-09-24T16:21:37Z")}, true},
		{"mount error from an earlier attempt", "Unprovisioned", []interface{}{condition("False", "Error", "2026-09-24T16:20:59Z")}, false},
		{"condition that succeeded", "Provisioned", []interface{}{condition("True", "", "2026-09-24T16:21:37Z")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := newDevice(map[string]interface{}{}, nil)
			status := map[string]interface{}{"provisionPhase": tc.phase}
			if tc.conditions != nil {
				status["conditions"] = tc.conditions
			}
			obj.Object["status"] = status
			phase, failure := blockDeviceProgress(obj, since)
			if phase != tc.phase {
				t.Errorf("phase = %q, want %q", phase, tc.phase)
			}
			if (failure != "") != tc.wantFailure {
				t.Errorf("failure = %q, want failure: %v", failure, tc.wantFailure)
			}
			if tc.wantFailure && !strings.HasPrefix(failure, "Mounted: failed to update device mount") {
				t.Errorf("failure message = %q", failure)
			}
		})
	}
}

func TestDiskDriverDiffSuppress(t *testing.T) {
	key := "disk_provisioner.0.longhorn.0.disk_driver"
	for engine, want := range map[string]bool{"LonghornV1": true, "LonghornV2": false} {
		d := resourceData(t, map[string]interface{}{
			"provision": true,
			"disk_provisioner": []interface{}{map[string]interface{}{
				"longhorn": []interface{}{map[string]interface{}{"engine_version": engine}},
			}},
		})
		if got := diskDriverDiffSuppress(key, "", "auto", d); got != want {
			t.Errorf("engine %s: suppress = %v, want %v", engine, got, want)
		}
	}
}

func TestFailureTracker(t *testing.T) {
	start := time.Date(2026, 9, 24, 16, 0, 0, 0, time.UTC)
	mount := "Mounted: wrong fs type, bad option, bad superblock on /dev/vdb"
	syncing := "AddedToNode: admission webhook \"validator.longhorn.io\" denied the request: spec and status of disks on node n1 are being syncing and please retry later."

	f := &failureTracker{}
	if f.fatal(start, mount) {
		t.Error("an error seen for the first time must not be final")
	}
	if f.fatal(start.Add(10*time.Second), mount) {
		t.Error("an error seen for 10 s must not be final")
	}
	if !f.fatal(start.Add(failureGrace), mount) {
		t.Error("an error that persists for the grace period must be final")
	}

	f = &failureTracker{}
	f.fatal(start, mount)
	f.fatal(start.Add(10*time.Second), "")
	if f.fatal(start.Add(failureGrace), mount) {
		t.Error("an error that cleared in between must start a new grace period")
	}

	f = &failureTracker{}
	for i := 0; i < 5; i++ {
		if f.fatal(start.Add(time.Duration(i)*failureGrace), syncing) {
			t.Fatal("an error asking to retry later must never be final")
		}
	}
}
