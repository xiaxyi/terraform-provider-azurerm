package servicebus

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/servicebus/mgmt/2021-06-01-preview/servicebus"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/resourcegroups"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/servicebus/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/servicebus/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceServiceBusNamespaceAuthorizationRule() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceServiceBusNamespaceAuthorizationRuleCreateUpdate,
		Read:   resourceServiceBusNamespaceAuthorizationRuleRead,
		Update: resourceServiceBusNamespaceAuthorizationRuleCreateUpdate,
		Delete: resourceServiceBusNamespaceAuthorizationRuleDelete,

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.NamespaceAuthorizationRuleID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		// function takes a schema map and adds the authorization rule properties to it
		Schema: resourceServiceBusNamespaceAuthorizationRuleSchema(),

		CustomizeDiff: pluginsdk.CustomizeDiffShim(authorizationRuleCustomizeDiff),
	}
}

func resourceServiceBusNamespaceAuthorizationRuleSchema() map[string]*pluginsdk.Schema {
	s := map[string]*pluginsdk.Schema{
		"name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validate.AuthorizationRuleName(),
		},
		//lintignore: S013
		"namespace_id": {
			Type:         pluginsdk.TypeString,
			Required:     features.ThreePointOhBeta(),
			Optional:     !features.ThreePointOhBeta(),
			Computed:     !features.ThreePointOhBeta(),
			ForceNew:     true,
			ValidateFunc: validate.NamespaceID,
			ConflictsWith: func() []string {
				if !features.ThreePointOhBeta() {
					return []string{"namespace_name", "resource_group_name"}
				}
				return []string{}
			}(),
		},
	}

	if !features.ThreePointOhBeta() {
		s["namespace_name"] = &pluginsdk.Schema{
			Type:          pluginsdk.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ValidateFunc:  validate.NamespaceName,
			Deprecated:    `Deprecated in favor of "namespace_id"`,
			ConflictsWith: []string{"namespace_id"},
		}

		s["resource_group_name"] = &pluginsdk.Schema{
			Type:          pluginsdk.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ValidateFunc:  resourcegroups.ValidateName,
			Deprecated:    `Deprecated in favor of "namespace_id"`,
			ConflictsWith: []string{"namespace_id"},
		}
	}

	return authorizationRuleSchemaFrom(s)
}

func resourceServiceBusNamespaceAuthorizationRuleCreateUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ServiceBus.NamespacesClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for ServiceBus Namespace Authorization Rule create/update.")

	var resourceId parse.NamespaceAuthorizationRuleId
	if namespaceIdLit := d.Get("namespace_id").(string); namespaceIdLit != "" {
		namespaceId, _ := parse.NamespaceID(namespaceIdLit)
		resourceId = parse.NewNamespaceAuthorizationRuleID(namespaceId.SubscriptionId, namespaceId.ResourceGroup, namespaceId.Name, d.Get("name").(string))
	} else if !features.ThreePointOhBeta() {
		resourceId = parse.NewNamespaceAuthorizationRuleID(subscriptionId, d.Get("resource_group_name").(string), d.Get("namespace_name").(string), d.Get("name").(string))
	}

	if d.IsNewResource() {
		existing, err := client.GetAuthorizationRule(ctx, resourceId.ResourceGroup, resourceId.NamespaceName, resourceId.AuthorizationRuleName)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing %s: %+v", resourceId, err)
			}
		}

		if !utils.ResponseWasNotFound(existing.Response) {
			return tf.ImportAsExistsError("azurerm_servicebus_namespace_authorization_rule", resourceId.ID())
		}
	}

	parameters := servicebus.SBAuthorizationRule{
		Name: utils.String(resourceId.AuthorizationRuleName),
		SBAuthorizationRuleProperties: &servicebus.SBAuthorizationRuleProperties{
			Rights: expandAuthorizationRuleRights(d),
		},
	}

	if _, err := client.CreateOrUpdateAuthorizationRule(ctx, resourceId.ResourceGroup, resourceId.NamespaceName, resourceId.AuthorizationRuleName, parameters); err != nil {
		return fmt.Errorf("creating/updating %s: %+v", resourceId, err)
	}

	d.SetId(resourceId.ID())

	if err := waitForPairedNamespaceReplication(ctx, meta, resourceId.ResourceGroup, resourceId.NamespaceName, d.Timeout(pluginsdk.TimeoutUpdate)); err != nil {
		return fmt.Errorf("waiting for replication to complete for Service Bus Namespace Disaster Recovery Configs (Namespace %q / Resource Group %q): %s", resourceId.NamespaceName, resourceId.ResourceGroup, err)
	}

	return resourceServiceBusNamespaceAuthorizationRuleRead(d, meta)
}

func resourceServiceBusNamespaceAuthorizationRuleRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ServiceBus.NamespacesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.NamespaceAuthorizationRuleID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.GetAuthorizationRule(ctx, id.ResourceGroup, id.NamespaceName, id.AuthorizationRuleName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	keysResp, err := client.ListKeys(ctx, id.ResourceGroup, id.NamespaceName, id.AuthorizationRuleName)
	if err != nil {
		return fmt.Errorf("listing keys for %s: %+v", id, err)
	}

	d.Set("name", id.AuthorizationRuleName)
	if !features.ThreePointOhBeta() {
		d.Set("namespace_name", id.NamespaceName)
		d.Set("resource_group_name", id.ResourceGroup)
	}
	d.Set("namespace_id", parse.NewNamespaceID(id.SubscriptionId, id.ResourceGroup, id.NamespaceName).ID())

	if properties := resp.SBAuthorizationRuleProperties; properties != nil {
		listen, send, manage := flattenAuthorizationRuleRights(properties.Rights)
		d.Set("manage", manage)
		d.Set("listen", listen)
		d.Set("send", send)
	}

	d.Set("primary_key", keysResp.PrimaryKey)
	d.Set("primary_connection_string", keysResp.PrimaryConnectionString)
	d.Set("secondary_key", keysResp.SecondaryKey)
	d.Set("secondary_connection_string", keysResp.SecondaryConnectionString)
	d.Set("primary_connection_string_alias", keysResp.AliasPrimaryConnectionString)
	d.Set("secondary_connection_string_alias", keysResp.AliasSecondaryConnectionString)

	return nil
}

func resourceServiceBusNamespaceAuthorizationRuleDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ServiceBus.NamespacesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.NamespaceAuthorizationRuleID(d.Id())
	if err != nil {
		return err
	}

	if _, err = client.DeleteAuthorizationRule(ctx, id.ResourceGroup, id.NamespaceName, id.AuthorizationRuleName); err != nil {
		return fmt.Errorf("deleting %s: %+v", id, err)
	}

	if err := waitForPairedNamespaceReplication(ctx, meta, id.ResourceGroup, id.NamespaceName, d.Timeout(pluginsdk.TimeoutUpdate)); err != nil {
		return fmt.Errorf("waiting for replication to complete for Service Bus Namespace Disaster Recovery Configs (Namespace %q / Resource Group %q): %s", id.NamespaceName, id.ResourceGroup, err)
	}

	return nil
}
