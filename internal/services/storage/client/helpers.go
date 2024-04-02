// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage" // nolint: staticcheck
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
)

var (
	storageAccountsCache = map[string]accountDetails{}

	accountsLock    = sync.RWMutex{}
	credentialsLock = sync.RWMutex{}
)

type EndpointType string

const (
	EndpointTypeBlob  = "blob"
	EndpointTypeDfs   = "dfs"
	EndpointTypeFile  = "file"
	EndpointTypeQueue = "queue"
	EndpointTypeTable = "table"
)

type accountDetails struct {
	ID            string
	Kind          storage.Kind
	Sku           *storage.Sku
	ResourceGroup string
	Properties    *storage.AccountProperties

	accountKey *string
	name       string
}

func (ad *accountDetails) AccountKey(ctx context.Context, client Client) (*string, error) {
	credentialsLock.Lock()
	defer credentialsLock.Unlock()

	if ad.accountKey != nil {
		return ad.accountKey, nil
	}

	log.Printf("[DEBUG] Cache Miss - looking up the account key for storage account %q..", ad.name)
	props, err := client.AccountsClient.ListKeys(ctx, ad.ResourceGroup, ad.name, storage.ListKeyExpandKerb)
	if err != nil {
		return nil, fmt.Errorf("listing Keys for Storage Account %q (Resource Group %q): %+v", ad.name, ad.ResourceGroup, err)
	}

	if props.Keys == nil || len(*props.Keys) == 0 || (*props.Keys)[0].Value == nil {
		return nil, fmt.Errorf("keys were nil for Storage Account %q (Resource Group %q): %+v", ad.name, ad.ResourceGroup, err)
	}

	keys := *props.Keys
	ad.accountKey = keys[0].Value

	// force-cache this
	storageAccountsCache[ad.name] = *ad

	return ad.accountKey, nil
}

func (ad *accountDetails) DataPlaneEndpoint(endpointType EndpointType) (*string, error) {
	if ad.Properties == nil {
		return nil, fmt.Errorf("storage account %q has no properties", ad.name)
	}
	if ad.Properties.PrimaryEndpoints == nil {
		return nil, fmt.Errorf("storage account %q has missing endpoints", ad.name)
	}

	var baseUri string

	switch endpointType {
	case EndpointTypeBlob:
		if ad.Properties.PrimaryEndpoints.Blob != nil {
			baseUri = strings.TrimSuffix(*ad.Properties.PrimaryEndpoints.Blob, "/")
		}
	case EndpointTypeDfs:
		if ad.Properties.PrimaryEndpoints.Dfs != nil {
			baseUri = strings.TrimSuffix(*ad.Properties.PrimaryEndpoints.Dfs, "/")
		}
	case EndpointTypeFile:
		if ad.Properties.PrimaryEndpoints.File != nil {
			baseUri = strings.TrimSuffix(*ad.Properties.PrimaryEndpoints.File, "/")
		}
	case EndpointTypeQueue:
		if ad.Properties.PrimaryEndpoints.Queue != nil {
			baseUri = strings.TrimSuffix(*ad.Properties.PrimaryEndpoints.Queue, "/")
		}
	case EndpointTypeTable:
		if ad.Properties.PrimaryEndpoints.Table != nil {
			baseUri = strings.TrimSuffix(*ad.Properties.PrimaryEndpoints.Table, "/")
		}
	default:
		return nil, fmt.Errorf("internal-error: unrecognised endpoint type %q when building storage client", endpointType)
	}

	if baseUri == "" {
		return nil, fmt.Errorf("determining storage account %s endpoint for : %q", endpointType, ad.name)
	}

	return &baseUri, nil
}

func (c Client) AddToCache(accountName string, props storage.Account) error {
	accountsLock.Lock()
	defer accountsLock.Unlock()

	account, err := populateAccountDetails(accountName, props)
	if err != nil {
		return err
	}

	storageAccountsCache[accountName] = *account

	return nil
}

func (c Client) RemoveAccountFromCache(accountName string) {
	accountsLock.Lock()
	delete(storageAccountsCache, accountName)
	accountsLock.Unlock()
}

func (c Client) FindAccount(ctx context.Context, accountName string) (*accountDetails, error) {
	accountsLock.Lock()
	defer accountsLock.Unlock()

	if existing, ok := storageAccountsCache[accountName]; ok {
		return &existing, nil
	}

	accountsPage, err := c.AccountsClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving storage accounts: %+v", err)
	}

	var accounts []storage.Account
	for accountsPage.NotDone() {
		accounts = append(accounts, accountsPage.Values()...)
		err = accountsPage.NextWithContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("retrieving next page of storage accounts: %+v", err)
		}
	}

	for _, v := range accounts {
		if v.Name == nil {
			continue
		}

		account, err := populateAccountDetails(*v.Name, v)
		if err != nil {
			return nil, err
		}

		storageAccountsCache[*v.Name] = *account
	}

	if existing, ok := storageAccountsCache[accountName]; ok {
		return &existing, nil
	}

	return nil, nil
}

func populateAccountDetails(accountName string, props storage.Account) (*accountDetails, error) {
	if props.ID == nil {
		return nil, fmt.Errorf("`id` was nil for Account %q", accountName)
	}

	accountId := *props.ID
	id, err := commonids.ParseStorageAccountID(accountId)
	if err != nil {
		return nil, fmt.Errorf("parsing %q as a Resource ID: %+v", accountId, err)
	}

	return &accountDetails{
		name:          accountName,
		ID:            accountId,
		Kind:          props.Kind,
		Sku:           props.Sku,
		ResourceGroup: id.ResourceGroupName,
		Properties:    props.AccountProperties,
	}, nil
}
