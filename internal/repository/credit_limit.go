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
)

type creditLimitRepository struct {
	db     *mysql.Client
	logger *zap.Logger
}

func NewCreditLimitRepository(db *mysql.Client, logger *zap.Logger) entity.CreditLimitRepository {
	return &creditLimitRepository{
		db:     db,
		logger: logger,
	}
}

func (r *creditLimitRepository) Create(ctx context.Context, limit *entity.CreditLimit) error {
	tr := otel.Tracer("repository.credit_limit")
	ctx, span := tr.Start(ctx, "Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("credit_limit.id", limit.ID.String()),
		attribute.String("customer.id", limit.CustomerID.String()),
		attribute.Int("tenor_month", limit.TenorMonth),
		attribute.Float64("limit_amount", limit.LimitAmount),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(limit).Error; err != nil {
			r.logger.Error("failed to create credit limit",
				zap.Error(err),
				zap.String("customer_id", limit.CustomerID.String()),
				zap.Int("tenor_month", limit.TenorMonth),
			)
			return fmt.Errorf("failed to create credit limit: %w", err)
		}
		return nil
	})
}

func (r *creditLimitRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditLimit, error) {
	tr := otel.Tracer("repository.credit_limit")
	ctx, span := tr.Start(ctx, "GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("credit_limit.id", id.String()))

	var limit entity.CreditLimit
	if err := r.db.WithContext(ctx).First(&limit, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get credit limit by id",
			zap.Error(err),
			zap.String("credit_limit_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get credit limit: %w", err)
	}

	return &limit, nil
}

func (r *creditLimitRepository) GetByCustomerIDAndTenor(ctx context.Context, customerID uuid.UUID, tenorMonth int) (*entity.CreditLimit, error) {
	tr := otel.Tracer("repository.credit_limit")
	ctx, span := tr.Start(ctx, "GetByCustomerIDAndTenor")
	defer span.End()

	span.SetAttributes(
		attribute.String("customer.id", customerID.String()),
		attribute.Int("tenor_month", tenorMonth),
	)

	var limit entity.CreditLimit
	if err := r.db.WithContext(ctx).
		Where("customer_id = ? AND tenor_month = ?", customerID, tenorMonth).
		First(&limit).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get credit limit by customer id and tenor",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
			zap.Int("tenor_month", tenorMonth),
		)
		return nil, fmt.Errorf("failed to get credit limit: %w", err)
	}

	return &limit, nil
}

func (r *creditLimitRepository) GetAllByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CreditLimit, error) {
	tr := otel.Tracer("repository.credit_limit")
	ctx, span := tr.Start(ctx, "GetAllByCustomerID")
	defer span.End()

	span.SetAttributes(attribute.String("customer.id", customerID.String()))

	var limits []entity.CreditLimit
	if err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("tenor_month ASC").
		Find(&limits).Error; err != nil {
		r.logger.Error("failed to get credit limits by customer id",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, fmt.Errorf("failed to get credit limits: %w", err)
	}

	return limits, nil
}

func (r *creditLimitRepository) UpdateUsedAmount(ctx context.Context, id uuid.UUID, amount float64) error {
	tr := otel.Tracer("repository.credit_limit")
	ctx, span := tr.Start(ctx, "UpdateUsedAmount")
	defer span.End()

	span.SetAttributes(
		attribute.String("credit_limit.id", id.String()),
		attribute.Float64("amount", amount),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		var limit entity.CreditLimit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&limit, "id = ?", id).Error; err != nil {
			r.logger.Error("failed to get credit limit for update",
				zap.Error(err),
				zap.String("credit_limit_id", id.String()),
			)
			return fmt.Errorf("failed to get credit limit for update: %w", err)
		}

		if limit.UsedAmount+amount > limit.LimitAmount {
			return fmt.Errorf("insufficient credit limit: available %.2f, requested %.2f",
				limit.LimitAmount-limit.UsedAmount, amount)
		}

		limit.UsedAmount += amount
		if err := tx.Save(&limit).Error; err != nil {
			r.logger.Error("failed to update credit limit used amount",
				zap.Error(err),
				zap.String("credit_limit_id", id.String()),
			)
			return fmt.Errorf("failed to update credit limit used amount: %w", err)
		}

		return nil
	})
}

func (r *creditLimitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tr := otel.Tracer("repository.credit_limit")
	ctx, span := tr.Start(ctx, "Delete")
	defer span.End()

	span.SetAttributes(attribute.String("credit_limit.id", id.String()))

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		var limit entity.CreditLimit
		if err := tx.First(&limit, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("credit limit not found")
			}
			r.logger.Error("failed to get credit limit for deletion",
				zap.Error(err),
				zap.String("credit_limit_id", id.String()),
			)
			return fmt.Errorf("failed to get credit limit for deletion: %w", err)
		}

		if limit.UsedAmount > 0 {
			return fmt.Errorf("cannot delete credit limit: limit is currently in use")
		}

		if err := tx.Delete(&limit).Error; err != nil {
			r.logger.Error("failed to delete credit limit",
				zap.Error(err),
				zap.String("credit_limit_id", id.String()),
			)
			return fmt.Errorf("failed to delete credit limit: %w", err)
		}

		return nil
	})
}
