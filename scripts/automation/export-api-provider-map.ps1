param(
    [ValidateSet("Csv", "Json")]
    [string] $Format = "Csv",

    [string] $OutputPath
)

$ErrorActionPreference = "Stop"
$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot "..\.."))

$documentationSets = @(
    @{ Path = "website/docs/r"; Type = "Resource" },
    @{ Path = "website/docs/d"; Type = "DataSource" },
    @{ Path = "website/docs/ephemeral-resources"; Type = "EphemeralResource" }
)

$ownershipRules = @(
    @{
        Pattern  = '^azurerm_key_vault(?:_|$)'
        Provider = 'Microsoft.KeyVault'
        Source   = 'ServiceFamily:azurerm_key_vault'
    }
)

$documents = foreach ($documentationSet in $documentationSets) {
    $documentationPath = Join-Path $repositoryRoot $documentationSet.Path
    if (-not (Test-Path $documentationPath)) {
        continue
    }

    foreach ($document in Get-ChildItem $documentationPath -Filter "*.html.markdown" -File) {
        $terraformName = "azurerm_$($document.Name -replace '\.html\.markdown$', '')"
        $relativePath = $document.FullName.Substring($repositoryRoot.Length + 1).Replace("\", "/")
        $lines = Get-Content $document.FullName
        $generatedProviders = @()
        $apiHeading = [Array]::IndexOf($lines, "## API Providers")

        if ($apiHeading -ge 0) {
            for ($lineNumber = $apiHeading + 1; $lineNumber -lt $lines.Count; $lineNumber++) {
                if ($lines[$lineNumber] -match "^## ") {
                    break
                }

                if ($lines[$lineNumber] -match '^\* `([^`]+)` - (.+)$') {
                    $generatedProviders += [pscustomobject]@{
                        Provider    = $Matches[1]
                        APIVersions = $Matches[2]
                    }
                }
            }
        }

        $importHeading = [Array]::IndexOf($lines, "## Import")
        $importLines = @()
        if ($importHeading -ge 0) {
            for ($lineNumber = $importHeading + 1; $lineNumber -lt $lines.Count; $lineNumber++) {
                if ($lines[$lineNumber] -match "^## ") {
                    break
                }

                $importLines += $lines[$lineNumber]
            }
        }

        $armImportProviders = @(
            [regex]::Matches(
                ($importLines -join "`n"),
                '(?i)/subscriptions/[^\s`"'']+/[^\s`"'']*providers/(Microsoft\.[A-Za-z0-9.]+)(?:/|\b)'
            ) |
                ForEach-Object { $_.Groups[1].Value } |
                Sort-Object -Unique
        )

        [pscustomobject]@{
            ResourceType       = $documentationSet.Type
            TerraformName      = $terraformName
            DocumentationPath  = $relativePath
            GeneratedProviders = $generatedProviders
            ARMImportProviders = $armImportProviders
        }
    }
}

$resourceOwnershipByName = @{}
$relationships = @()

foreach ($document in $documents | Where-Object ResourceType -eq "Resource") {
    $owningProvider = ""
    $ownershipSource = "Unmapped"

    if ($document.ARMImportProviders.Count -eq 1) {
        $owningProvider = $document.ARMImportProviders[0]
        $ownershipSource = "ARMImportID"
    } else {
        $ownershipRule = $ownershipRules | Where-Object {
            $document.TerraformName -match $_.Pattern
        } | Select-Object -First 1

        if ($null -ne $ownershipRule) {
            $owningProvider = $ownershipRule.Provider
            $ownershipSource = $ownershipRule.Source
        } elseif ($document.GeneratedProviders.Count -eq 1) {
            $owningProvider = $document.GeneratedProviders[0].Provider
            $ownershipSource = "SingleGeneratedAPIProvider"
        } elseif ($document.ARMImportProviders.Count -gt 1) {
            $ownershipSource = "AmbiguousARMImportID"
        } elseif ($document.GeneratedProviders.Count -gt 1) {
            $ownershipSource = "AmbiguousGeneratedAPIProviders"
        }
    }

    if ($owningProvider -ne "") {
        $resourceOwnershipByName[$document.TerraformName] = [pscustomobject]@{
            Provider = $owningProvider
            Source   = $ownershipSource
        }
    }

    $ownerVersions = @(
        $document.GeneratedProviders |
            Where-Object Provider -eq $owningProvider |
            ForEach-Object APIVersions |
            Sort-Object -Unique
    )

    $relationships += [pscustomobject]@{
        ResourceType      = $document.ResourceType
        TerraformName     = $document.TerraformName
        OwningProvider    = $owningProvider
        OwnerAPIVersions  = $ownerVersions -join ", "
        OwnershipSource   = $ownershipSource
        APIProviders      = ($document.GeneratedProviders.Provider | Sort-Object -Unique) -join "; "
        DocumentationPath = $document.DocumentationPath
    }
}

foreach ($document in $documents | Where-Object ResourceType -ne "Resource") {
    $owningProvider = ""
    $ownershipSource = "Unmapped"
    $ownershipRule = $ownershipRules | Where-Object {
        $document.TerraformName -match $_.Pattern
    } | Select-Object -First 1

    if ($null -ne $ownershipRule) {
        $owningProvider = $ownershipRule.Provider
        $ownershipSource = $ownershipRule.Source
    } elseif ($resourceOwnershipByName.ContainsKey($document.TerraformName)) {
        $resourceOwnership = $resourceOwnershipByName[$document.TerraformName]
        $owningProvider = $resourceOwnership.Provider
        $ownershipSource = "MatchingResource:$($resourceOwnership.Source)"
    } elseif ($document.GeneratedProviders.Count -eq 1) {
        $owningProvider = $document.GeneratedProviders[0].Provider
        $ownershipSource = "SingleGeneratedAPIProvider"
    } elseif ($document.GeneratedProviders.Count -gt 1) {
        $ownershipSource = "AmbiguousGeneratedAPIProviders"
    }

    $ownerVersions = @(
        $document.GeneratedProviders |
            Where-Object Provider -eq $owningProvider |
            ForEach-Object APIVersions |
            Sort-Object -Unique
    )

    $relationships += [pscustomobject]@{
        ResourceType      = $document.ResourceType
        TerraformName     = $document.TerraformName
        OwningProvider    = $owningProvider
        OwnerAPIVersions  = $ownerVersions -join ", "
        OwnershipSource   = $ownershipSource
        APIProviders      = ($document.GeneratedProviders.Provider | Sort-Object -Unique) -join "; "
        DocumentationPath = $document.DocumentationPath
    }
}

$relationships = @($relationships | Sort-Object ResourceType, TerraformName)

if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    $extension = if ($Format -eq "Json") { "json" } else { "csv" }
    $OutputPath = Join-Path $repositoryRoot "azurerm-resource-ownership-map.$extension"
} elseif (-not [System.IO.Path]::IsPathRooted($OutputPath)) {
    $OutputPath = Join-Path (Get-Location) $OutputPath
}

if ($Format -eq "Json") {
    $relationships | ConvertTo-Json -Depth 3 | Set-Content $OutputPath
} else {
    $relationships | Export-Csv $OutputPath -NoTypeInformation
}

$mappedRelationships = @($relationships | Where-Object { $_.OwningProvider -ne "" })
$unmappedDocuments = @($relationships | Where-Object { $_.OwningProvider -eq "" })

Write-Host "Exported $($relationships.Count) ownership rows to $OutputPath"
Write-Host "Mapped Terraform names: $($mappedRelationships.Count)"
Write-Host "Owning providers: $(($mappedRelationships.OwningProvider | Sort-Object -Unique).Count)"
Write-Host "Unmapped or ambiguous documentation pages: $($unmappedDocuments.Count)"