package cacher

import (
	"fmt"
	"github.com/google/uuid"
)

const (
	cachePrefix       = "cache:object"
	customerPrefix    = "customer"
	documentPrefix    = "document"
	limitPrefix       = "credit_limit"
	transactionPrefix = "transaction"
)

func createCacheKey(key string) string {
	return key
}

func GetCustomerCacheKeyByID(id uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:id:%s", cachePrefix, customerPrefix, id.String()))
}

func GetCustomerCacheKeyByNIK(nik string) string {
	return createCacheKey(fmt.Sprintf("%s:%s:nik:%s", cachePrefix, customerPrefix, nik))
}

func GetCustomerDocumentsCacheKey(customerID uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:customer:%s:all", cachePrefix, documentPrefix, customerID.String()))
}

func GetCustomerDocumentCacheKey(documentID uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:id:%s", cachePrefix, documentPrefix, documentID.String()))
}

func GetCustomerCreditLimitsCacheKey(customerID uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:customer:%s:all", cachePrefix, limitPrefix, customerID.String()))
}

func GetCreditLimitCacheKey(limitID uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:id:%s", cachePrefix, limitPrefix, limitID.String()))
}

func GetCustomerTransactionsCacheKey(customerID uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:customer:%s:all", cachePrefix, transactionPrefix, customerID.String()))
}

func GetTransactionCacheKey(transactionID uuid.UUID) string {
	return createCacheKey(fmt.Sprintf("%s:%s:id:%s", cachePrefix, transactionPrefix, transactionID.String()))
}

func GetMultipleCustomerCacheKeys(ids []uuid.UUID) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = GetCustomerCacheKeyByID(id)
	}
	return keys
}

func GetMultipleCreditLimitCacheKeys(ids []uuid.UUID) []string {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = GetCreditLimitCacheKey(id)
	}
	return keys
}

func GetAssetCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("asset:%s", id.String())
}
