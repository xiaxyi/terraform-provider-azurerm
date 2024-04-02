package helpers

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-sdk/resource-manager/web/2023-01-01/webapps"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
	"github.com/tombuildsstuff/kermit/sdk/web/2022-09-01/web"
)

type Registry struct {
	Server   string `tfschema:"registry_server_url"`
	UserName string `tfschema:"registry_username"`
	Password string `tfschema:"registry_password"`
	Identity string `tfschema:"identity"`
}

func RegistrySchemaLinuxFunctionAppOnContainer() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		MaxItems: 1,
		Required: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"registry_server_url": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
					Description:  "The login endpoint of the container Registry url",
				},

				"registry_username": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					RequiredWith: []string{"registry.0.registry_password"},
					Description:  "The username to use for this Container Registry.",
				},

				"registry_password": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					RequiredWith: []string{"registry.0.registry_username"},
					Description:  "The name of the Secret Reference containing the password value for this user on the Container Registry.",
				},
			},
		},
	}
}

type SiteConfigLinuxFunctionAppOnContainer struct {
	AppInsightsConnectionString string `tfschema:"application_insights_connection_string"`
	Use32BitWorker              bool   `tfschema:"use_32_bit_worker"`
	FtpsState                   string `tfschema:"ftps_state"`
}

func SiteConfigSchemaLinuxFunctionAppOnContainer() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"application_insights_connection_string": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					Sensitive:    true,
					ValidateFunc: validation.StringIsNotEmpty,
					Description:  "The Connection String for linking the Linux Function App to Application Insights.",
				},

				"use_32_bit_worker": {
					Type:        pluginsdk.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Should the Linux Web App use a 32-bit worker.",
				},

				"ftps_state": {
					Type:     pluginsdk.TypeString,
					Optional: true,
					Default:  string(web.FtpsStateDisabled),
					ValidateFunc: validation.StringInSlice([]string{
						string(web.FtpsStateAllAllowed),
						string(web.FtpsStateDisabled),
						string(web.FtpsStateFtpsOnly),
					}, false),
					Description: "State of FTP / FTPS service for this function app. Possible values include: `AllAllowed`, `FtpsOnly` and `Disabled`. Defaults to `Disabled`.",
				},
			},
		},
	}
}

func ExpandSiteConfigLinuxFunctionAppOnContainer(siteConfig []SiteConfigLinuxFunctionAppOnContainer, existing *webapps.SiteConfig, metadata sdk.ResourceMetaData, registry Registry, version string, storageString string) (*webapps.SiteConfig, error) {
	if len(siteConfig) == 0 {
		return nil, nil
	}

	expanded := &webapps.SiteConfig{}
	if existing != nil {
		expanded = existing
		// need to zero fxversion to re-calculate based on changes below or removing app_stack doesn't apply
		expanded.LinuxFxVersion = utils.String("")
	}

	appSettings := make([]webapps.NameValuePair, 0)
	if existing != nil && existing.AppSettings != nil {
		appSettings = *existing.AppSettings
	}

	appSettings = updateOrAppendAppSettings(appSettings, "FUNCTIONS_EXTENSION_VERSION", version, false)
	appSettings = updateOrAppendAppSettings(appSettings, "AzureWebJobsStorage", storageString, false)

	linuxFunctionOnContainerSiteConfig := siteConfig[0]
	if linuxFunctionOnContainerSiteConfig.AppInsightsConnectionString == "" {
		appSettings = updateOrAppendAppSettings(appSettings, "APPLICATIONINSIGHTS_CONNECTION_STRING", linuxFunctionOnContainerSiteConfig.AppInsightsConnectionString, true)
	} else {
		appSettings = updateOrAppendAppSettings(appSettings, "APPLICATIONINSIGHTS_CONNECTION_STRING", linuxFunctionOnContainerSiteConfig.AppInsightsConnectionString, false)
	}

	// update docker related settings
	appSettings = updateOrAppendAppSettings(appSettings, "DOCKER_REGISTRY_SERVER_URL", registry.Server, false)
	if registry.UserName != "" {
		appSettings = updateOrAppendAppSettings(appSettings, "DOCKER_REGISTRY_SERVER_USERNAME", registry.UserName, false)
	}
	if registry.Password != "" {
		appSettings = updateOrAppendAppSettings(appSettings, "DOCKER_REGISTRY_SERVER_PASSWORD", registry.Password, false)
	}

	expanded.AppSettings = &appSettings

	if metadata.ResourceData.HasChange("site_config.0.ftps_state") {
		expanded.FtpsState = pointer.To(webapps.FtpsState(linuxFunctionOnContainerSiteConfig.FtpsState))
	}
	expanded.Use32BitWorkerProcess = utils.Bool(linuxFunctionOnContainerSiteConfig.Use32BitWorker)
	return expanded, nil
}

func FlattenSiteConfigLinuxFunctionAppOnContainer(functionAppOnContainer *webapps.SiteConfig) (*SiteConfigLinuxFunctionAppOnContainer, error) {
	result := &SiteConfigLinuxFunctionAppOnContainer{
		Use32BitWorker: pointer.From(functionAppOnContainer.Use32BitWorkerProcess),
		FtpsState:      string(pointer.From(functionAppOnContainer.FtpsState)),
	}

	return result, nil
}

func EncodeLinuxFunctionAppOnContainerLinuxFxVersion(input []Registry, image string) *string {
	if len(input) == 0 {
		return utils.String("")
	}
	dockerUrl := input[0].Server
	httpPrefixes := []string{"https://", "http://"}
	for _, prefix := range httpPrefixes {
		dockerUrl = strings.TrimPrefix(dockerUrl, prefix)
	}
	return utils.String(fmt.Sprintf("DOCKER|%s/%s", dockerUrl, image))
}

func DecodeLinuxFunctionAppOnContainerLinuxFxVersion(input *string) (string, error) {
	if input == nil {
		return "", nil
	}
	parts := strings.Split(*input, "|")
	value := parts[1]
	if len(parts) != 2 || parts[0] != "DOCKER" {
		return "", fmt.Errorf("unrecognised LinuxFxVersion format received, got %s", input)
	}

	return value[strings.Index(parts[1], "/")+1:], nil
}
