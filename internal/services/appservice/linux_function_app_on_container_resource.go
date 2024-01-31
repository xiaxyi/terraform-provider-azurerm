package appservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/go-azure-sdk/resource-manager/containerapps/2023-05-01/managedenvironments"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/appservice/helpers"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/appservice/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/appservice/validate"
	kvValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/validate"
	storageValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/storage/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
	"github.com/tombuildsstuff/kermit/sdk/web/2022-09-01/web"
)

type LinuxFunctionAppOnContainerResource struct{}

type LinuxFunctionAppOnContainerModel struct {
	Name                 string `tfschema:"name"`
	ResourceGroup        string `tfschema:"resource_group_name"`
	ManagedEnvironmentId string `tfschema:"container_app_environment_id"`

	StorageAccountName      string `tfschema:"storage_account_name"`
	StorageAccountKey       string `tfschema:"storage_account_access_key"`
	StorageKeyVaultSecretID string `tfschema:"storage_key_vault_secret_id"`

	BuiltinLogging            bool              `tfschema:"builtin_logging_enabled"`
	AppSettings               map[string]string `tfschema:"app_settings"`
	FunctionExtensionsVersion string            `tfschema:"functions_extension_version"`

	Registries     []helpers.Registry                              `tfschema:"registry"`
	ContainerImage string                                          `tfschema:"container_image"`
	SiteConfig     []helpers.SiteConfigLinuxFunctionAppOnContainer `tfschema:"site_config"`
	Tags           map[string]string                               `tfschema:"tags"`
}

var _ sdk.ResourceWithUpdate = LinuxFunctionAppOnContainerResource{}

func (r LinuxFunctionAppOnContainerResource) ModelObject() interface{} {
	return &LinuxFunctionAppOnContainerModel{}
}

func (r LinuxFunctionAppOnContainerResource) ResourceType() string {
	return "azurerm_linux_function_app_on_container"
}

func (r LinuxFunctionAppOnContainerResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return validate.FunctionAppID
}

func (r LinuxFunctionAppOnContainerResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validate.ContainerizedFunctionAppName,
			Description:  "Specifies the name of the Function App.",
		},

		"resource_group_name": commonschema.ResourceGroupName(),

		"container_app_environment_id": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: managedenvironments.ValidateManagedEnvironmentID,
			Description:  "The ID of the Container App Environment to host this Container App.",
		},

		"storage_account_name": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: storageValidate.StorageAccountName,
			Description:  "The backend storage account name which will be used by this Function App.",
			ExactlyOneOf: []string{
				"storage_account_name",
				"storage_key_vault_secret_id",
			},
		},

		"storage_account_access_key": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			Sensitive:    true,
			ValidateFunc: validation.NoZeroValues,
			ConflictsWith: []string{
				"storage_key_vault_secret_id",
			},
			Description: "The access key which will be used to access the storage account for the Function App.",
		},

		"storage_key_vault_secret_id": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: kvValidate.NestedItemIdWithOptionalVersion,
			ExactlyOneOf: []string{
				"storage_account_name",
				"storage_key_vault_secret_id",
			},
			Description: "The Key Vault Secret ID, including version, that contains the Connection String to connect to the storage account for this Function App.",
		},

		"builtin_logging_enabled": {
			Type:        pluginsdk.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Should built in logging be enabled. Configures `AzureWebJobsDashboard` app setting based on the configured storage setting",
		},

		"functions_extension_version": {
			Type:        pluginsdk.TypeString,
			Optional:    true,
			Default:     "~4",
			Description: "The runtime version associated with the Function App.",
		},

		"app_settings": {
			Type:     pluginsdk.TypeMap,
			Optional: true,
			Elem: &pluginsdk.Schema{
				Type: pluginsdk.TypeString,
			},
			Description: "A map of key-value pairs for [App Settings](https://docs.microsoft.com/en-us/azure/azure-functions/functions-app-settings) and custom values.",
		},

		"registry": helpers.RegistrySchemaLinuxFunctionAppOnContainer(),

		"container_image": {
			Type:        pluginsdk.TypeString,
			Required:    true,
			Description: "The name of the Container Image that includes image tag",
		},

		"site_config": helpers.SiteConfigSchemaLinuxFunctionAppOnContainer(),

		"tags": tags.Schema(),
	}

}

func (r LinuxFunctionAppOnContainerResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"kind": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"default_hostname": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"outbound_ip_addresses": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},
	}
}

func (r LinuxFunctionAppOnContainerResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			storageDomainSuffix, ok := metadata.Client.Account.Environment.Storage.DomainSuffix()
			if !ok {
				return fmt.Errorf("could not determine Storage domain suffix for environment %q", metadata.Client.Account.Environment.Name)
			}
			var linuxFunctionAppOnContainer LinuxFunctionAppOnContainerModel

			if err := metadata.Decode(&linuxFunctionAppOnContainer); err != nil {
				return err
			}

			client := metadata.Client.AppService.WebAppsClient
			containerEnvClient := metadata.Client.ContainerApps.ManagedEnvironmentClient
			subscriptionId := metadata.Client.Account.SubscriptionId

			id := parse.NewFunctionAppID(subscriptionId, linuxFunctionAppOnContainer.ResourceGroup, linuxFunctionAppOnContainer.Name)

			existing, err := client.Get(ctx, id.ResourceGroup, id.SiteName)
			if err != nil && !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing Linux %s: %+v", id, err)
			}

			if !utils.ResponseWasNotFound(existing.Response) {
				return metadata.ResourceRequiresImport(r.ResourceType(), id)
			}

			availabilityRequest := web.ResourceNameAvailabilityRequest{
				Name: utils.String(linuxFunctionAppOnContainer.Name),
				Type: web.CheckNameResourceTypesMicrosoftWebsites,
			}

			envId, err := managedenvironments.ParseManagedEnvironmentID(linuxFunctionAppOnContainer.ManagedEnvironmentId)
			if err != nil {
				return fmt.Errorf("parsing Container App Environment ID for %s: %+v", id, err)
			}

			env, err := containerEnvClient.Get(ctx, *envId)
			if err != nil {
				return fmt.Errorf("reading %s for %s: %+v", *envId, id, err)
			}

			checkName, err := client.CheckNameAvailability(ctx, availabilityRequest)
			if err != nil {
				return fmt.Errorf("checking name availability for Linux %s: %+v", id, err)
			}
			if checkName.NameAvailable != nil && !*checkName.NameAvailable {
				return fmt.Errorf("the Site Name %q failed the availability check: %+v", id.SiteName, *checkName.Message)
			}

			// storage using MSI is currently not supported in function on container.
			storageString := linuxFunctionAppOnContainer.StorageAccountName
			if linuxFunctionAppOnContainer.StorageKeyVaultSecretID != "" {
				storageString = fmt.Sprintf(helpers.StorageStringFmtKV, linuxFunctionAppOnContainer.StorageKeyVaultSecretID)
			} else {
				storageString = fmt.Sprintf(helpers.StorageStringFmt, linuxFunctionAppOnContainer.StorageAccountName, linuxFunctionAppOnContainer.StorageAccountKey, *storageDomainSuffix)
			}

			if linuxFunctionAppOnContainer.BuiltinLogging {
				if linuxFunctionAppOnContainer.AppSettings == nil {
					linuxFunctionAppOnContainer.AppSettings = make(map[string]string)
				}
				linuxFunctionAppOnContainer.AppSettings["AzureWebJobsDashboard"] = storageString
			}

			siteConfig, err := helpers.ExpandSiteConfigLinuxFunctionAppOnContainer(linuxFunctionAppOnContainer.SiteConfig, nil, metadata, linuxFunctionAppOnContainer.Registries[0], linuxFunctionAppOnContainer.FunctionExtensionsVersion, storageString)
			siteConfig.LinuxFxVersion = helpers.EncodeLinuxFunctionAppOnContainerLinuxFxVersion(linuxFunctionAppOnContainer.Registries, linuxFunctionAppOnContainer.ContainerImage)

			siteEnvelope := web.Site{
				Tags:     tags.FromTypedObject(linuxFunctionAppOnContainer.Tags),
				Location: utils.String(location.Normalize(env.Model.Location)),
				Kind:     utils.String("functionapp,linux,container,azurecontainerapps"),
				SiteProperties: &web.SiteProperties{
					SiteConfig:           siteConfig,
					ManagedEnvironmentID: pointer.To(linuxFunctionAppOnContainer.ManagedEnvironmentId),
				},
			}
			siteConfig.AppSettings = helpers.MergeUserAppSettings(siteConfig.AppSettings, linuxFunctionAppOnContainer.AppSettings)

			js, _ := json.Marshal(siteEnvelope)
			log.Printf("DDDDD after%s", js)

			future, err := client.CreateOrUpdate(ctx, id.ResourceGroup, id.SiteName, siteEnvelope)
			if err != nil {
				return fmt.Errorf("creating Linux %s: %+v", id, err)
			}

			if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
				return fmt.Errorf("waiting for creation of Linux %s: %+v", id, err)
			}

			metadata.SetID(id)
			return nil
		},
	}
}

func (r LinuxFunctionAppOnContainerResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.AppService.WebAppsClient
			id, err := parse.FunctionAppID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			functionAppOnContainer, err := client.Get(ctx, id.ResourceGroup, id.SiteName)
			if err != nil {
				if utils.ResponseWasNotFound(functionAppOnContainer.Response) {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("reading Linux %s: %+v", id, err)
			}

			if functionAppOnContainer.SiteProperties == nil {
				return fmt.Errorf("reading properties of Linux %s: %+v", id, err)
			}

			props := *functionAppOnContainer.SiteProperties

			appSettingsResp, err := client.ListApplicationSettings(ctx, id.ResourceGroup, id.SiteName)
			if err != nil {
				return fmt.Errorf("reading App Settings for Linux %s: %+v", id, err)
			}

			state := LinuxFunctionAppOnContainerModel{
				Name:                 id.SiteName,
				ResourceGroup:        id.ResourceGroup,
				ManagedEnvironmentId: pointer.From(props.ManagedEnvironmentID),
				Tags:                 tags.ToTypedObject(functionAppOnContainer.Tags),
			}

			configResp, err := client.GetConfiguration(ctx, id.ResourceGroup, id.SiteName)
			if err != nil {
				return fmt.Errorf("making Read request on AzureRM Function App Configuration %q: %+v", id.SiteName, err)
			}

			siteConfig, err := helpers.FlattenSiteConfigLinuxFunctionAppOnContainer(configResp.SiteConfig)
			state.SiteConfig = []helpers.SiteConfigLinuxFunctionAppOnContainer{*siteConfig}

			state.ContainerImage, err = helpers.DecodeLinuxFunctionAppOnContainerLinuxFxVersion(configResp.LinuxFxVersion)
			if err != nil {
				return fmt.Errorf("flattening linuxFxVersion: %s", err)
			}

			js, _ := json.Marshal(appSettingsResp)
			log.Printf("DDDDD APPSETTING RESPONSE %s", js)

			state.unpackLinuxFunctionAppOnContainerAppSettings(appSettingsResp)

			return nil
		},
	}
}

func (r LinuxFunctionAppOnContainerResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.AppService.WebAppsClient
			id, err := parse.FunctionAppID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}
			metadata.Logger.Infof("deleting Linux %s", *id)

			deleteMetrics := true
			deleteEmptyServerFarm := false
			if _, err := client.Delete(ctx, id.ResourceGroup, id.SiteName, &deleteMetrics, &deleteEmptyServerFarm); err != nil {
				return fmt.Errorf("deleting Linux %s: %+v", id, err)
			}
			return nil
		},
	}
}

func (r LinuxFunctionAppOnContainerResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			storageDomainSuffix, ok := metadata.Client.Account.Environment.Storage.DomainSuffix()
			if !ok {
				return fmt.Errorf("could not determine Storage domain suffix for environment %q", metadata.Client.Account.Environment.Name)
			}

			client := metadata.Client.AppService.WebAppsClient
			id, err := parse.FunctionAppID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			// need to set this property to true as the config got written to state despite the update action actually failed.
			metadata.ResourceData.Partial(true)

			var state LinuxFunctionAppOnContainerModel
			if err := metadata.Decode(&state); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			existing, err := client.Get(ctx, id.ResourceGroup, id.SiteName)
			if err != nil {
				return fmt.Errorf("reading Linux %s: %v", id, err)
			}

			storageString := state.StorageAccountName
			if state.StorageKeyVaultSecretID != "" {
				storageString = fmt.Sprintf(helpers.StorageStringFmtKV, state.StorageKeyVaultSecretID)
			} else {
				storageString = fmt.Sprintf(helpers.StorageStringFmt, state.StorageAccountName, state.StorageAccountKey, *storageDomainSuffix)
			}

			siteConfig, err := helpers.ExpandSiteConfigLinuxFunctionAppOnContainer(state.SiteConfig, existing.SiteConfig, metadata, state.Registries[0], state.FunctionExtensionsVersion, storageString)
			if metadata.ResourceData.HasChange("site_config") {
				existing.SiteConfig = siteConfig
			}

			if metadata.ResourceData.HasChange("registry") || metadata.ResourceData.HasChange("container_image") {
				existing.SiteConfig.LinuxFxVersion = helpers.EncodeLinuxFunctionAppOnContainerLinuxFxVersion(state.Registries, state.ContainerImage)
			}

			existing.SiteConfig.AppSettings = helpers.MergeUserAppSettings(siteConfig.AppSettings, state.AppSettings)

			if metadata.ResourceData.HasChange("tags") {
				existing.Tags = tags.FromTypedObject(state.Tags)
			}
			updateFuture, err := client.CreateOrUpdate(ctx, id.ResourceGroup, id.SiteName, existing)
			if err != nil {
				return fmt.Errorf("updating Linux %s: %+v", id, err)
			}
			if err := updateFuture.WaitForCompletionRef(ctx, client.Client); err != nil {
				return fmt.Errorf("waiting to update %s: %+v", id, err)
			}

			if _, err := client.UpdateConfiguration(ctx, id.ResourceGroup, id.SiteName, web.SiteConfigResource{SiteConfig: existing.SiteConfig}); err != nil {
				return fmt.Errorf("updating Site Config for Linux %s: %s", id, err)
			}
			return nil
		},
	}
}

func (m *LinuxFunctionAppOnContainerModel) unpackLinuxFunctionAppOnContainerAppSettings(input web.StringDictionary) {
	if input.Properties == nil {
		return
	}

	appSettings := make(map[string]string)

	var dockerSettings helpers.Registry
	m.BuiltinLogging = false

	for k, v := range input.Properties {
		switch k {
		case "FUNCTIONS_EXTENSION_VERSION":
			m.FunctionExtensionsVersion = pointer.From(v)
		case "DOCKER_REGISTRY_SERVER_URL":
			dockerSettings.Server = pointer.From(v)
		case "DOCKER_REGISTRY_SERVER_USERNAME":
			dockerSettings.UserName = utils.NormalizeNilableString(v)
		case "DOCKER_REGISTRY_SERVER_PASSWORD":
			dockerSettings.Password = utils.NormalizeNilableString(v)
		case "APPLICATIONINSIGHTS_CONNECTION_STRING":
			m.SiteConfig[0].AppInsightsConnectionString = utils.NormalizeNilableString(v)
		case "AzureWebJobsStorage":
			if v != nil && strings.HasPrefix(*v, "@Microsoft.KeyVault") {
				trimmed := strings.TrimPrefix(strings.TrimSuffix(*v, ")"), "@Microsoft.KeyVault(SecretUri=")
				m.StorageKeyVaultSecretID = trimmed
			} else {
				m.StorageAccountName, m.StorageAccountKey = helpers.ParseWebJobsStorageString(v)
			}
		default:
			appSettings[k] = utils.NormalizeNilableString(v)
		}
	}
	m.AppSettings = appSettings
}
