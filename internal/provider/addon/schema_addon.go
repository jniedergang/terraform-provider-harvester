package addon

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/harvester/terraform-provider-harvester/internal/util"
	"github.com/harvester/terraform-provider-harvester/pkg/constants"
)

func Schema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		constants.FieldAddonEnabled: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		constants.FieldAddonValuesContent: {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		constants.FieldAddonRepo: {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Helm repository URL for the addon chart. Override to install custom or experimental addons.",
		},
		constants.FieldAddonChart: {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Helm chart name for the addon. Override to install custom or experimental addons.",
		},
		constants.FieldAddonVersion: {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Helm chart version for the addon. Override to install a specific version.",
		},
	}
	util.NamespacedSchemaWrap(s, true)
	return s
}

func DataSourceSchema() map[string]*schema.Schema {
	return util.DataSourceSchemaWrap(Schema())
}
