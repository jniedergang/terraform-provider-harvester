package blockdevice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harvester/harvester/pkg/builder"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/harvester/terraform-provider-harvester/internal/config"
	"github.com/harvester/terraform-provider-harvester/internal/util"
	"github.com/harvester/terraform-provider-harvester/pkg/constants"
	"github.com/harvester/terraform-provider-harvester/pkg/helper"
	"github.com/harvester/terraform-provider-harvester/pkg/importer"
)

var blockDeviceGVR = k8sschema.GroupVersionResource{
	Group:    "harvesterhci.io",
	Version:  "v1beta1",
	Resource: "blockdevices",
}

func ResourceBlockDevice() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBlockDeviceCreate,
		ReadContext:   resourceBlockDeviceRead,
		UpdateContext: resourceBlockDeviceUpdate,
		DeleteContext: resourceBlockDeviceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: Schema(),
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(10 * time.Minute),
			Read:    schema.DefaultTimeout(2 * time.Minute),
			Update:  schema.DefaultTimeout(10 * time.Minute),
			Delete:  schema.DefaultTimeout(10 * time.Minute),
			Default: schema.DefaultTimeout(2 * time.Minute),
		},
	}
}

// resourceBlockDeviceCreate adopts an existing block device by applying spec updates.
func resourceBlockDeviceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace := d.Get(constants.FieldCommonNamespace).(string)
	name := d.Get(constants.FieldCommonName).(string)

	obj, err := c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(fmt.Errorf("block device %s/%s not found, it must already exist on the node: %w", namespace, name, err))
	}

	applyBlockDeviceSpec(d, obj)

	since := time.Now().Truncate(time.Second)
	if _, err = c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		return diag.FromErr(err)
	}

	// Track the device before waiting, so that a failed provisioning can still
	// be unprovisioned by terraform destroy.
	d.SetId(helper.BuildID(namespace, name))
	obj, err = waitForProvisionPhase(ctx, c.DynamicClient, namespace, name, targetPhase(d), since, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(resourceBlockDeviceImport(d, obj))
}

func resourceBlockDeviceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := helper.IDParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	obj, err := c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(resourceBlockDeviceImport(d, obj))
}

func resourceBlockDeviceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := helper.IDParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	obj, err := c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	applyBlockDeviceSpec(d, obj)

	since := time.Now().Truncate(time.Second)
	if _, err = c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		return diag.FromErr(err)
	}

	obj, err = waitForProvisionPhase(ctx, c.DynamicClient, namespace, name, targetPhase(d), since, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(resourceBlockDeviceImport(d, obj))
}

// resourceBlockDeviceDelete deprovisions the device but does NOT delete the K8s object.
func resourceBlockDeviceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := helper.IDParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	obj, err := c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if unprovisionSpec(obj) {
		since := time.Now().Truncate(time.Second)
		_, err = c.DynamicClient.Resource(blockDeviceGVR).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return diag.FromErr(err)
		}
		if err == nil {
			if _, err = waitForProvisionPhase(ctx, c.DynamicClient, namespace, name, phaseUnprovisioned, since, d.Timeout(schema.TimeoutDelete)); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.SetId("")
	return nil
}

// unprovisionSpec only flips spec.provision to false and reports whether a
// change is needed. The provisioner and device tags are kept on purpose: the
// node disk manager needs them to unprovision, and its webhook silently
// restores the previous spec when an update drops the provisioner of a
// provisioned device.
func unprovisionSpec(obj *unstructured.Unstructured) bool {
	provision, _, _ := unstructured.NestedBool(obj.Object, "spec", "provision")
	if !provision {
		return false
	}
	_ = unstructured.SetNestedField(obj.Object, false, "spec", "provision")
	return true
}

const (
	phaseProvisioned   = "Provisioned"
	phaseUnprovisioned = "Unprovisioned"
)

func targetPhase(d *schema.ResourceData) string {
	if d.Get(constants.FieldBlockDeviceProvision).(bool) {
		return phaseProvisioned
	}
	return phaseUnprovisioned
}

// blockDeviceProgress returns the provision phase of the device and, when the
// node disk manager reported a failed step at or after `since`, its message.
func blockDeviceProgress(obj *unstructured.Unstructured, since time.Time) (string, string) {
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "provisionPhase")
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, raw := range conditions {
		condition, ok := raw.(map[string]interface{})
		if !ok || condition["status"] != "False" {
			continue
		}
		if reason, _ := condition["reason"].(string); reason != "Error" && reason != "Failed" {
			continue
		}
		updated, _ := condition["lastUpdateTime"].(string)
		at, err := time.Parse(time.RFC3339, updated)
		if err != nil || at.Before(since) {
			continue
		}
		kind, _ := condition["type"].(string)
		message, _ := condition["message"].(string)
		return phase, fmt.Sprintf("%s: %s", kind, strings.TrimSpace(message))
	}
	return phase, ""
}

// failureGrace is how long an error reported by the node disk manager must
// persist before it is considered final: it retries on its own, and some
// errors (Longhorn syncing the node disks) clear on the next attempt.
const failureGrace = 30 * time.Second

// failureTracker decides when a reported error is final: it must not ask to
// retry later and must persist for failureGrace.
type failureTracker struct {
	firstSeen time.Time
}

func (f *failureTracker) fatal(now time.Time, failure string) bool {
	if failure == "" || strings.Contains(failure, "retry later") {
		f.firstSeen = time.Time{}
		return false
	}
	if f.firstSeen.IsZero() {
		f.firstSeen = now
	}
	return now.Sub(f.firstSeen) >= failureGrace
}

// waitForProvisionPhase waits until the device reaches the target phase, and
// stops early when the node disk manager keeps reporting the same kind of
// error for this change (for example a new disk without a filesystem that is
// provisioned without force_formatted).
func waitForProvisionPhase(ctx context.Context, client dynamic.Interface, namespace, name, target string, since time.Time, timeout time.Duration) (*unstructured.Unstructured, error) {
	deadline := time.Now().Add(timeout)
	tracker := &failureTracker{}
	for {
		obj, err := client.Resource(blockDeviceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) && target == phaseUnprovisioned {
				return nil, nil
			}
			return nil, err
		}
		phase, failure := blockDeviceProgress(obj, since)
		if phase == target {
			return obj, nil
		}
		if tracker.fatal(time.Now(), failure) {
			return nil, fmt.Errorf("block device %s/%s did not reach %s: %s", namespace, name, target, failure)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out after %s waiting for block device %s/%s to reach %s (current phase %q)", timeout, namespace, name, target, phase)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func resourceBlockDeviceImport(d *schema.ResourceData, obj *unstructured.Unstructured) error {
	stateGetter, err := importer.ResourceBlockDeviceStateGetter(obj)
	if err != nil {
		return err
	}
	return util.ResourceStatesSet(d, stateGetter)
}

// applyBlockDeviceSpec applies user-specified spec fields to the unstructured object.
func applyBlockDeviceSpec(d *schema.ResourceData, obj *unstructured.Unstructured) {
	provision := d.Get(constants.FieldBlockDeviceProvision).(bool)
	_ = unstructured.SetNestedField(obj.Object, provision, "spec", "provision")

	forceFormatted := d.Get(constants.FieldBlockDeviceForceFormatted).(bool)
	_ = unstructured.SetNestedField(obj.Object, forceFormatted, "spec", "fileSystem", "forceFormatted")

	// Device tags (spec.tags)
	if v, ok := d.GetOk(constants.FieldBlockDeviceDeviceTags); ok {
		rawTags := v.([]interface{})
		tags := make([]interface{}, len(rawTags))
		copy(tags, rawTags)
		_ = unstructured.SetNestedSlice(obj.Object, tags, "spec", "tags")
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", "tags")
	}

	// Provisioner block. It is never removed from the object: the node disk
	// manager needs it to unprovision the device, and its webhook restores the
	// previous spec when an update drops it. Without a configured block, a
	// device to provision gets a Longhorn V1 provisioner, like in the UI.
	if v, ok := d.GetOk(constants.FieldBlockDeviceProvisioner); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		applyProvisioner(v.([]interface{})[0].(map[string]interface{}), obj)
	} else if _, found, _ := unstructured.NestedMap(obj.Object, "spec", "provisioner"); provision && !found {
		_ = unstructured.SetNestedField(obj.Object, map[string]interface{}{
			"longhorn": map[string]interface{}{"engineVersion": engineLonghornV1},
		}, "spec", "provisioner")
	}

	// Apply description annotation
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	if desc, ok := d.GetOk(constants.FieldCommonDescription); ok {
		annotations["field.cattle.io/description"] = desc.(string)
	} else {
		delete(annotations, "field.cattle.io/description")
	}
	obj.SetAnnotations(annotations)

	// Tags and user labels are owned by the configuration: stale ones are
	// removed, labels managed by the node disk manager or Harvester are kept.
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	for key := range labels {
		isTag := strings.HasPrefix(key, builder.LabelPrefixHarvesterTag) || strings.HasPrefix(key, importer.LegacyBlockDeviceTagPrefix)
		if isTag || !importer.IsBlockDeviceSystemLabel(key) {
			delete(labels, key)
		}
	}
	for key, value := range d.Get(constants.FieldCommonTags).(map[string]interface{}) {
		labels[builder.LabelPrefixHarvesterTag+key] = value.(string)
	}
	for key, value := range d.Get(constants.FieldCommonLabels).(map[string]interface{}) {
		labels[key] = value.(string)
	}
	obj.SetLabels(labels)
}

func applyProvisioner(provMap map[string]interface{}, obj *unstructured.Unstructured) {
	if lhList, ok := provMap[constants.FieldBlockDeviceProvisionerLonghorn]; ok {
		lhItems := lhList.([]interface{})
		if len(lhItems) > 0 && lhItems[0] != nil {
			lh := lhItems[0].(map[string]interface{})
			engine, _ := lh[constants.FieldBlockDeviceProvisionerLonghornEV].(string)
			if engine == "" {
				engine = engineLonghornV1
			}
			longhorn := map[string]interface{}{"engineVersion": engine}
			if driver, _ := lh[constants.FieldBlockDeviceProvisionerLonghornDD].(string); driver != "" && engine == engineLonghornV2 {
				longhorn["diskDriver"] = driver
			}
			_ = unstructured.SetNestedField(obj.Object, map[string]interface{}{"longhorn": longhorn}, "spec", "provisioner")
			return
		}
	}

	if lvmList, ok := provMap[constants.FieldBlockDeviceProvisionerLVM]; ok {
		lvmItems := lvmList.([]interface{})
		if len(lvmItems) > 0 && lvmItems[0] != nil {
			lvm := lvmItems[0].(map[string]interface{})
			provisionerMap := map[string]interface{}{
				"lvm": map[string]interface{}{
					"vgName": lvm[constants.FieldBlockDeviceProvisionerLVMVGName],
				},
			}
			if params, ok := lvm[constants.FieldBlockDeviceProvisionerLVMParameters]; ok {
				paramsList := params.([]interface{})
				if len(paramsList) > 0 {
					provisionerMap["lvm"].(map[string]interface{})["parameters"] = paramsList
				}
			}
			_ = unstructured.SetNestedField(obj.Object, provisionerMap, "spec", "provisioner")
		}
	}
}
