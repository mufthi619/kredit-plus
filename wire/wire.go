//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	"kredit-plus/infra/mysql"
	"kredit-plus/infra/redis"
	"kredit-plus/internal/handler"
	"kredit-plus/internal/repository"
	"kredit-plus/internal/service"
)

var (
	AssetSet = wire.NewSet(
		repository.NewAssetRepository,
		service.NewAssetService,
		handler.NewAssetHandler,
	)

	CustomerSet = wire.NewSet(
		repository.NewCustomerRepository,
		service.NewCustomerService,
		handler.NewCustomerHandler,
	)

	CreditLimitSet = wire.NewSet(
		repository.NewCreditLimitRepository,
		service.NewCreditLimitService,
		handler.NewCreditLimitHandler,
	)

	TransactionProviderSet = wire.NewSet(
		repository.NewTransactionRepository,
		repository.NewCustomerRepository,
		repository.NewCreditLimitRepository,
		repository.NewAssetRepository,
		service.NewTransactionService,
		handler.NewTransactionHandler,
	)

	DomainSet = wire.NewSet(
		AssetSet,
		CustomerSet,
		CreditLimitSet,
		TransactionProviderSet,
	)
)

func InitializeAssetHandler(
	db *mysql.Client,
	redisClient *redis.Client,
	logger *zap.Logger,
) (*handler.AssetHandler, error) {
	wire.Build(AssetSet)
	return &handler.AssetHandler{}, nil
}

func InitializeCustomerHandler(
	db *mysql.Client,
	redisClient *redis.Client,
	logger *zap.Logger,
) (*handler.CustomerHandler, error) {
	wire.Build(CustomerSet)
	return &handler.CustomerHandler{}, nil
}

func InitializeCreditLimitHandler(
	db *mysql.Client,
	redisClient *redis.Client,
	logger *zap.Logger,
) (*handler.CreditLimitHandler, error) {
	wire.Build(CreditLimitSet)
	return &handler.CreditLimitHandler{}, nil
}

func InitializeTransactionProviderHandler(
	db *mysql.Client,
	redisClient *redis.Client,
	logger *zap.Logger,
) (*handler.TransactionHandler, error) {
	wire.Build(TransactionProviderSet)
	return &handler.TransactionHandler{}, nil
}
