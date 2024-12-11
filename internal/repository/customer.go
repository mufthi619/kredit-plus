package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"kredit-plus/cacher"
	"kredit-plus/infra/mysql"
	"kredit-plus/infra/redis"
	"kredit-plus/internal/entity"
)

type customerRepository struct {
	db     *mysql.Client
	redis  *redis.Client
	logger *zap.Logger
}

func NewCustomerRepository(db *mysql.Client, redisClient *redis.Client, logger *zap.Logger) entity.CustomerRepository {
	return &customerRepository{
		db:     db,
		redis:  redisClient,
		logger: logger,
	}
}

func (r *customerRepository) Create(ctx context.Context, customer *entity.Customer) error {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("customer.id", customer.ID.String()),
		attribute.String("customer.nik", customer.NIK),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(customer).Error; err != nil {
			r.logger.Error("failed to create customer",
				zap.Error(err),
				zap.String("customer_id", customer.ID.String()),
			)
			return fmt.Errorf("failed to create customer: %w", err)
		}
		return nil
	})
}

func (r *customerRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error) {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("customer.id", id.String()))

	cacheKey := cacher.GetCustomerCacheKeyByID(id)
	var customer entity.Customer
	cachedData, err := r.redis.Get(ctx, cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &customer); err == nil {
			return &customer, nil
		}
	}

	if err := r.db.WithContext(ctx).
		Preload("Documents").
		First(&customer, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get customer by id",
			zap.Error(err),
			zap.String("customer_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if customerJSON, err := json.Marshal(customer); err == nil {
		if err := r.redis.Set(ctx, cacheKey, string(customerJSON), entity.DefaultCacheTTL); err != nil {
			r.logger.Warn("failed to cache customer",
				zap.Error(err),
				zap.String("customer_id", id.String()),
			)
		}
	}

	return &customer, nil
}

func (r *customerRepository) GetByNIK(ctx context.Context, nik string) (*entity.Customer, error) {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "GetByNIK")
	defer span.End()

	span.SetAttributes(attribute.String("customer.nik", nik))

	cacheKey := cacher.GetCustomerCacheKeyByNIK(nik)
	var customer entity.Customer
	cachedData, err := r.redis.Get(ctx, cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &customer); err == nil {
			return &customer, nil
		}
	}

	if err := r.db.WithContext(ctx).
		Preload("Documents").
		First(&customer, "nik = ?", nik).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get customer by nik",
			zap.Error(err),
			zap.String("nik", nik),
		)
		return nil, fmt.Errorf("failed to get customer by nik: %w", err)
	}

	if customerJSON, err := json.Marshal(customer); err == nil {
		if err := r.redis.Set(ctx, cacheKey, string(customerJSON), entity.DefaultCacheTTL); err != nil {
			r.logger.Warn("failed to cache customer",
				zap.Error(err),
				zap.String("nik", nik),
			)
		}
	}

	return &customer, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *entity.Customer) error {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "Update")
	defer span.End()

	span.SetAttributes(
		attribute.String("customer.id", customer.ID.String()),
		attribute.String("customer.nik", customer.NIK),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Save(customer).Error; err != nil {
			r.logger.Error("failed to update customer",
				zap.Error(err),
				zap.String("customer_id", customer.ID.String()),
			)
			return fmt.Errorf("failed to update customer: %w", err)
		}

		cacheKeys := []string{
			cacher.GetCustomerCacheKeyByID(customer.ID),
			cacher.GetCustomerCacheKeyByNIK(customer.NIK),
			cacher.GetCustomerDocumentsCacheKey(customer.ID),
		}

		for _, key := range cacheKeys {
			if err := r.redis.Del(ctx, key); err != nil {
				r.logger.Warn("failed to invalidate customer cache",
					zap.Error(err),
					zap.String("cache_key", key),
				)
			}
		}

		return nil
	})
}

func (r *customerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "Delete")
	defer span.End()

	span.SetAttributes(attribute.String("customer.id", id.String()))

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		var customer entity.Customer
		if err := tx.First(&customer, "id = ?", id).Error; err != nil {
			r.logger.Error("failed to get customer for deletion",
				zap.Error(err),
				zap.String("customer_id", id.String()),
			)
			return fmt.Errorf("failed to get customer for deletion: %w", err)
		}

		if err := tx.Model(&entity.Customer{}).
			Where("id = ?", id).
			Update("is_active", false).Error; err != nil {
			r.logger.Error("failed to delete customer",
				zap.Error(err),
				zap.String("customer_id", id.String()),
			)
			return fmt.Errorf("failed to delete customer: %w", err)
		}

		cacheKeys := []string{
			cacher.GetCustomerCacheKeyByID(id),
			cacher.GetCustomerCacheKeyByNIK(customer.NIK),
			cacher.GetCustomerDocumentsCacheKey(id),
			cacher.GetCustomerCreditLimitsCacheKey(id),
			cacher.GetCustomerTransactionsCacheKey(id),
		}

		if err := r.redis.Del(ctx, cacheKeys...); err != nil {
			r.logger.Warn("failed to invalidate customer related caches",
				zap.Error(err),
				zap.String("customer_id", id.String()),
				zap.Strings("cache_keys", cacheKeys),
			)
		}

		return nil
	})
}

func (r *customerRepository) CreateDocument(ctx context.Context, doc *entity.CustomerDocument) error {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "CreateDocument")
	defer span.End()

	span.SetAttributes(
		attribute.String("customer.id", doc.CustomerID.String()),
		attribute.String("document.type", string(doc.DocumentType)),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			r.logger.Error("failed to create customer document",
				zap.Error(err),
				zap.String("customer_id", doc.CustomerID.String()),
				zap.String("document_type", string(doc.DocumentType)),
			)
			return fmt.Errorf("failed to create customer document: %w", err)
		}

		cacheKeys := []string{
			cacher.GetCustomerCacheKeyByID(doc.CustomerID),
			cacher.GetCustomerDocumentsCacheKey(doc.CustomerID),
			cacher.GetCustomerDocumentCacheKey(doc.ID),
		}

		if err := r.redis.Del(ctx, cacheKeys...); err != nil {
			r.logger.Warn("failed to invalidate customer document related caches",
				zap.Error(err),
				zap.String("customer_id", doc.CustomerID.String()),
				zap.String("document_id", doc.ID.String()),
				zap.Strings("cache_keys", cacheKeys),
			)
		}

		return nil
	})
}

func (r *customerRepository) GetDocuments(ctx context.Context, filter entity.DocumentFilterRepository) (documents []entity.CustomerDocument, count int64, err error) {
	tr := otel.Tracer("repository.customer")
	ctx, span := tr.Start(ctx, "GetDocuments")
	defer span.End()

	span.SetAttributes(
		attribute.String("customer.id", filter.CustomerID.String()),
		attribute.Int("limit", filter.Limit),
		attribute.Int("offset", filter.Offset),
	)

	if filter.Limit < 0 || filter.Offset < 0 {
		r.logger.Error("invalid pagination parameters",
			zap.Int("limit", filter.Limit),
			zap.Int("offset", filter.Offset),
			zap.String("customer_id", filter.CustomerID.String()),
		)
		return nil, 0, fmt.Errorf("invalid pagination parameters: limit and offset must be non-negative")
	}

	query := r.db.WithContext(ctx).Model(&entity.CustomerDocument{}).
		Where("customer_id = ?", filter.CustomerID).
		Order("created_at DESC")
	if filter.DocumentType != nil {
		query = query.Where("document_type = ?", *filter.DocumentType)
		span.SetAttributes(attribute.String("document.type", string(*filter.DocumentType)))
	}

	if err := query.Count(&count).Error; err != nil {
		r.logger.Error("failed to count customer documents",
			zap.Error(err),
			zap.String("customer_id", filter.CustomerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to count customer documents: %w", err)
	}

	if count > 0 && filter.Offset >= int(count) {
		r.logger.Warn("offset exceeds total count",
			zap.Int("offset", filter.Offset),
			zap.Int64("total_count", count),
			zap.String("customer_id", filter.CustomerID.String()),
		)
		return []entity.CustomerDocument{}, count, nil
	}

	if err := query.
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&documents).Error; err != nil {
		r.logger.Error("failed to get customer documents",
			zap.Error(err),
			zap.String("customer_id", filter.CustomerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to get customer documents: %w", err)
	}

	return documents, count, nil
}