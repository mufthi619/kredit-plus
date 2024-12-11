package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"kredit-plus/infra/mysql"
	"kredit-plus/internal/entity"
	"time"
)

type transactionRepository struct {
	db     *mysql.Client
	logger *zap.Logger
}

func NewTransactionRepository(db *mysql.Client, logger *zap.Logger) entity.TransactionRepository {
	return &transactionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *transactionRepository) Create(ctx context.Context, transaction *entity.Transaction) error {
	tr := otel.Tracer("repository.transaction")
	ctx, span := tr.Start(ctx, "Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("transaction.id", transaction.ID.String()),
		attribute.String("customer.id", transaction.CustomerID.String()),
		attribute.String("asset.id", transaction.AssetID.String()),
		attribute.String("contract.number", transaction.ContractNumber),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		var creditLimit entity.CreditLimit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("customer_id = ? AND tenor_month = ?", transaction.CustomerID, transaction.TenorMonth).
			First(&creditLimit).Error; err != nil {
			r.logger.Error("failed to get credit limit for transaction",
				zap.Error(err),
				zap.String("customer_id", transaction.CustomerID.String()),
			)
			return fmt.Errorf("failed to get credit limit: %w", err)
		}

		if creditLimit.UsedAmount+transaction.OTRAmount > creditLimit.LimitAmount {
			return fmt.Errorf("insufficient credit limit: available %.2f, requested %.2f",
				creditLimit.LimitAmount-creditLimit.UsedAmount, transaction.OTRAmount)
		}

		if err := tx.Create(transaction).Error; err != nil {
			r.logger.Error("failed to create transaction",
				zap.Error(err),
				zap.String("customer_id", transaction.CustomerID.String()),
			)
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		installments := r.generateInstallments(transaction)
		if err := tx.Create(&installments).Error; err != nil {
			r.logger.Error("failed to create transaction details",
				zap.Error(err),
				zap.String("transaction_id", transaction.ID.String()),
			)
			return fmt.Errorf("failed to create transaction details: %w", err)
		}

		creditLimit.UsedAmount += transaction.OTRAmount
		if err := tx.Save(&creditLimit).Error; err != nil {
			r.logger.Error("failed to update credit limit",
				zap.Error(err),
				zap.String("credit_limit_id", creditLimit.ID.String()),
			)
			return fmt.Errorf("failed to update credit limit: %w", err)
		}

		return nil
	})
}

func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	tr := otel.Tracer("repository.transaction")
	ctx, span := tr.Start(ctx, "GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("transaction.id", id.String()))

	var transaction entity.Transaction
	if err := r.db.WithContext(ctx).
		Preload("TransactionDetail").
		Preload("Customer").
		Preload("Asset").
		First(&transaction, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get transaction by id",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

func (r *transactionRepository) GetByContractNumber(ctx context.Context, contractNumber string) (*entity.Transaction, error) {
	tr := otel.Tracer("repository.transaction")
	ctx, span := tr.Start(ctx, "GetByContractNumber")
	defer span.End()

	span.SetAttributes(attribute.String("contract.number", contractNumber))

	var transaction entity.Transaction
	if err := r.db.WithContext(ctx).
		Preload("TransactionDetail").
		Preload("Customer").
		Preload("Asset").
		First(&transaction, "contract_number = ?", contractNumber).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get transaction by contract number",
			zap.Error(err),
			zap.String("contract_number", contractNumber),
		)
		return nil, fmt.Errorf("failed to get transaction by contract number: %w", err)
	}

	return &transaction, nil
}

func (r *transactionRepository) GetAllByCustomerID(ctx context.Context, customerID uuid.UUID, filter entity.TransactionFilterRepository) ([]entity.Transaction, int64, error) {
	tr := otel.Tracer("repository.transaction")
	ctx, span := tr.Start(ctx, "GetAllByCustomerID")
	defer span.End()

	span.SetAttributes(
		attribute.String("customer.id", customerID.String()),
		attribute.String("status", string(filter.Status)),
		attribute.Int("limit", filter.Limit),
		attribute.Int("offset", filter.Offset),
	)

	var transactions []entity.Transaction
	var count int64

	query := r.db.WithContext(ctx).Model(&entity.Transaction{}).
		Where("customer_id = ?", customerID)
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&count).Error; err != nil {
		r.logger.Error("failed to count customer transactions",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	if err := query.
		Preload("TransactionDetail").
		Preload("Asset").
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&transactions).Error; err != nil {
		r.logger.Error("failed to get customer transactions",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, count, nil
}

func (r *transactionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.TransactionStatus) error {
	tr := otel.Tracer("repository.transaction")
	ctx, span := tr.Start(ctx, "UpdateStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("transaction.id", id.String()),
		attribute.String("status", string(status)),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		var transaction entity.Transaction
		if err := tx.First(&transaction, "id = ?", id).Error; err != nil {
			r.logger.Error("failed to get transaction for status update",
				zap.Error(err),
				zap.String("transaction_id", id.String()),
			)
			return fmt.Errorf("failed to get transaction: %w", err)
		}

		if err := tx.Model(&transaction).Update("status", status).Error; err != nil {
			r.logger.Error("failed to update transaction status",
				zap.Error(err),
				zap.String("transaction_id", id.String()),
			)
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})
}

func (r *transactionRepository) generateInstallments(transaction *entity.Transaction) []entity.TransactionDetail {
	installments := make([]entity.TransactionDetail, transaction.TenorMonth)
	installmentAmount := transaction.InstallmentAmount
	dueDate := time.Now().UTC()

	for i := 0; i < transaction.TenorMonth; i++ {
		dueDate = dueDate.AddDate(0, 1, 0)
		installments[i] = entity.TransactionDetail{
			ID:                uuid.New(),
			TransactionID:     transaction.ID,
			InstallmentNumber: i + 1,
			Amount:            installmentAmount,
			DueDate:           dueDate,
			Status:            entity.TransactionDetailStatusPending,
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		}
	}

	return installments
}
