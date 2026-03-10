package addon

import (
	"context"
	"fmt"
	"time"

	harvsterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harvester/terraform-provider-harvester/internal/config"
	"github.com/harvester/terraform-provider-harvester/internal/util"
	"github.com/harvester/terraform-provider-harvester/pkg/client"
	"github.com/harvester/terraform-provider-harvester/pkg/constants"
	"github.com/harvester/terraform-provider-harvester/pkg/helper"
	"github.com/harvester/terraform-provider-harvester/pkg/importer"
)

func ResourceAddon() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAddonCreate,
		ReadContext:   resourceAddonRead,
		UpdateContext: resourceAddonUpdate,
		DeleteContext: resourceAddonDelete,
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

// The addon cannot be created. It can only be updated (enabled/configured).
func resourceAddonCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}
	namespace := d.Get(constants.FieldCommonNamespace).(string)
	name := d.Get(constants.FieldCommonName).(string)
	obj, err := c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	if diags := updateAddon(ctx, c, d, namespace, obj); diags.HasError() {
		return diags
	}
	if err := waitForAddonReady(ctx, c, d, namespace, name, schema.TimeoutCreate); err != nil {
		return diag.FromErr(err)
	}
	return resourceAddonRead(ctx, d, meta)
}

func resourceAddonRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}
	namespace, name, err := helper.IDParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	obj, err := c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return diag.FromErr(resourceAddonImport(d, obj))
}

func resourceAddonUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}
	namespace, name, err := helper.IDParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	obj, err := c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if diags := updateAddon(ctx, c, d, namespace, obj); diags.HasError() {
		return diags
	}
	if err := waitForAddonReady(ctx, c, d, namespace, name, schema.TimeoutUpdate); err != nil {
		return diag.FromErr(err)
	}
	return resourceAddonRead(ctx, d, meta)
}

// The addon cannot be deleted. It can only be disabled.
func resourceAddonDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := meta.(*config.Config).K8sClient()
	if err != nil {
		return diag.FromErr(err)
	}
	namespace, name, err := helper.IDParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	obj, err := c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	objCopy := obj.DeepCopy()
	objCopy.Spec.Enabled = false
	objCopy.Spec.ValuesContent = ""
	_, err = c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Update(ctx, objCopy, metav1.UpdateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func updateAddon(ctx context.Context, c *client.Client, d *schema.ResourceData, namespace string, oldAddon *harvsterv1.Addon) diag.Diagnostics {
	toUpdate, err := util.ResourceConstruct(d, Updater(oldAddon))
	if err != nil {
		return diag.FromErr(err)
	}
	newAddon := toUpdate.(*harvsterv1.Addon)
	newAddon, err = c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Update(ctx, newAddon, metav1.UpdateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(resourceAddonImport(d, newAddon))
}

func waitForAddonReady(ctx context.Context, c *client.Client, d *schema.ResourceData, namespace, name, timeoutKey string) error {
	enabled := d.Get(constants.FieldAddonEnabled).(bool)
	if !enabled {
		return nil
	}
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(harvsterv1.AddonEnabling),
			string(harvsterv1.AddonUpdating),
			string(harvsterv1.AddonInitState),
		},
		Target: []string{
			string(harvsterv1.AddonDeployed),
		},
		Refresh:    addonStateRefresh(ctx, c, namespace, name),
		Timeout:    d.Timeout(timeoutKey),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err := stateConf.WaitForStateContext(ctx)
	return err
}

func addonStateRefresh(ctx context.Context, c *client.Client, namespace, name string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		obj, err := c.HarvesterClient.HarvesterhciV1beta1().Addons(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
		state := string(obj.Status.Status)
		if state == "" {
			state = string(harvsterv1.AddonInitState)
		}
		// Check for operation failure
		if harvsterv1.AddonOperationFailed.IsTrue(obj) {
			return obj, state, fmt.Errorf("addon %s/%s deployment failed", namespace, name)
		}
		return obj, state, nil
	}
}

func resourceAddonImport(d *schema.ResourceData, obj *harvsterv1.Addon) error {
	stateGetter, err := importer.ResourceAddonStateGetter(obj)
	if err != nil {
		return err
	}
	return util.ResourceStatesSet(d, stateGetter)
}
