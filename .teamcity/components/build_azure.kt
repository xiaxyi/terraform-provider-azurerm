import jetbrains.buildServer.configs.kotlin.v2019_2.ParametrizedWithType

class ClientConfiguration(var clientId: String,
                          var clientSecret: String,
                          val subscriptionId : String,
                          val tenantId : String,
                          val clientIdAlt: String,
                          val clientSecretAlt: String,
                          val subscriptionIdAlt : String) {
}

class LocationConfiguration(var primary : String, var secondary : String, var ternary : String, var rotate : Boolean) {
}

fun ParametrizedWithType.ConfigureAzureSpecificTestParameters(environment: String, config: ClientConfiguration, locationsForEnv: LocationConfiguration, useAltSubscription: Boolean = false) {
    hiddenPasswordVariable("env.ARM_CLIENT_ID", config.clientId, "The ID of the Service Principal used for Testing")
    hiddenPasswordVariable("env.ARM_CLIENT_ID_ALT", config.clientIdAlt, "The ID of the Alternate Service Principal used for Testing")
    hiddenPasswordVariable("env.ARM_CLIENT_SECRET", config.clientSecret, "The Client Secret of the Service Principal used for Testing")
    hiddenPasswordVariable("env.ARM_CLIENT_SECRET_ALT", config.clientSecretAlt, "The Client Secret of the Alternate Service Principal used for Testing")
    hiddenVariable("env.ARM_ENVIRONMENT", environment, "The Azure Environment in which the tests are running")
    hiddenVariable("env.ARM_PROVIDER_DYNAMIC_TEST", "%b".format(locationsForEnv.rotate), "Should tests rotate between the supported regions?")
    if (useAltSubscription) {
        hiddenPasswordVariable("env.ARM_SUBSCRIPTION_ID", config.subscriptionIdAlt, "The ID of the Azure Subscription used for Testing (swapped)")
        hiddenPasswordVariable("env.ARM_SUBSCRIPTION_ID_ALT", config.subscriptionId, "The ID of the Alternate Azure Subscription used for Testing (swapped)")
    } else {
        hiddenPasswordVariable("env.ARM_SUBSCRIPTION_ID", config.subscriptionId, "The ID of the Azure Subscription used for Testing")
        hiddenPasswordVariable("env.ARM_SUBSCRIPTION_ID_ALT", config.subscriptionIdAlt, "The ID of the Alternate Azure Subscription used for Testing")
    }
    hiddenPasswordVariable("env.ARM_TENANT_ID", config.tenantId, "The ID of the Azure Tenant used for Testing")
    hiddenVariable("env.ARM_TEST_LOCATION", locationsForEnv.primary, "The Primary region which should be used for testing")
    hiddenVariable("env.ARM_TEST_LOCATION_ALT", locationsForEnv.secondary, "The Secondary region which should be used for testing")
    hiddenVariable("env.ARM_TEST_LOCATION_ALT2", locationsForEnv.ternary, "The Tertiary region which should be used for testing")
    hiddenVariable("env.ARM_THREEPOINTZERO_BETA", "true", "Opt into the use of 3.0 beta resources")
}