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

type assetRepository struct {
	db     *mysql.Client
	redis  *redis.Client
	logger *zap.Logger
}

func NewAssetRepository(db *mysql.Client, redisClient *redis.Client, logger *zap.Logger) entity.AssetRepository {
	return &assetRepository{
		db:     db,
		redis:  redisClient,
		logger: logger,
	}
}

func (r *assetRepository) Create(ctx context.Context, asset *entity.Asset) error {
	tr := otel.Tracer("repository.asset")
	ctx, span := tr.Start(ctx, "Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("asset.id", asset.ID.String()),
		attribute.String("asset.name", asset.Name),
		attribute.String("asset.category", asset.Category),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(asset).Error; err != nil {
			r.logger.Error("failed to create asset",
				zap.Error(err),
				zap.String("asset_id", asset.ID.String()),
			)
			return fmt.Errorf("failed to create asset: %w", err)
		}
		return nil
	})
}

func (r *assetRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Asset, error) {
	tr := otel.Tracer("repository.asset")
	ctx, span := tr.Start(ctx, "GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("asset.id", id.String()))

	cacheKey := cacher.GetAssetCacheKey(id)
	var asset entity.Asset

	cachedData, err := r.redis.Get(ctx, cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &asset); err == nil {
			return &asset, nil
		}
	}

	if err := r.db.WithContext(ctx).First(&asset, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Error("failed to get asset by id",
			zap.Error(err),
			zap.String("asset_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	if assetJSON, err := json.Marshal(asset); err == nil {
		if err := r.redis.Set(ctx, cacheKey, string(assetJSON), entity.DefaultCacheTTL); err != nil {
			r.logger.Warn("failed to cache asset",
				zap.Error(err),
				zap.String("asset_id", id.String()),
			)
		}
	}

	return &asset, nil
}

func (r *assetRepository) GetAllWithFilter(ctx context.Context, filter entity.AssetFilterRepository) (assets []entity.Asset, count int64, err error) {
	tr := otel.Tracer("repository.asset")
	ctx, span := tr.Start(ctx, "List")
	defer span.End()

	span.SetAttributes(
		attribute.String("filter.category", filter.Category),
		attribute.Float64("filter.min_price", filter.MinPrice),
		attribute.Float64("filter.max_price", filter.MaxPrice),
		attribute.Int("filter.limit", filter.Limit),
		attribute.Int("filter.offset", filter.Offset),
	)

	if filter.Limit < 0 || filter.Offset < 0 {
		return nil, 0, fmt.Errorf("invalid pagination parameters: limit and offset must be non-negative")
	}

	query := r.db.WithContext(ctx).Model(&entity.Asset{})
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.MinPrice > 0 {
		query = query.Where("price >= ?", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		query = query.Where("price <= ?", filter.MaxPrice)
	}

	if err = query.Count(&count).Error; err != nil {
		r.logger.Error("failed to count assets",
			zap.Error(err),
			zap.Any("filter", filter),
		)
		return nil, 0, fmt.Errorf("failed to count assets: %w", err)
	}

	if count > 0 && filter.Offset >= int(count) {
		return []entity.Asset{}, count, nil
	}

	if err = query.
		Limit(filter.Limit).
		Offset(filter.Offset).
		Order("created_at DESC").
		Find(&assets).Error; err != nil {
		r.logger.Error("failed to list assets",
			zap.Error(err),
			zap.Any("filter", filter),
		)
		return nil, 0, fmt.Errorf("failed to list assets: %w", err)
	}

	return assets, count, nil
}

func (r *assetRepository) Update(ctx context.Context, asset *entity.Asset) error {
	tr := otel.Tracer("repository.asset")
	ctx, span := tr.Start(ctx, "Update")
	defer span.End()

	span.SetAttributes(
		attribute.String("asset.id", asset.ID.String()),
		attribute.String("asset.name", asset.Name),
	)

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Save(asset).Error; err != nil {
			r.logger.Error("failed to update asset",
				zap.Error(err),
				zap.String("asset_id", asset.ID.String()),
			)
			return fmt.Errorf("failed to update asset: %w", err)
		}

		cacheKey := cacher.GetAssetCacheKey(asset.ID)
		if err := r.redis.Del(ctx, cacheKey); err != nil {
			r.logger.Warn("failed to invalidate asset cache",
				zap.Error(err),
				zap.String("cache_key", cacheKey),
			)
		}

		return nil
	})
}

func (r *assetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tr := otel.Tracer("repository.asset")
	ctx, span := tr.Start(ctx, "Delete")
	defer span.End()

	span.SetAttributes(attribute.String("asset.id", id.String()))

	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		var asset entity.Asset
		if err := tx.First(&asset, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("asset not found")
			}
			r.logger.Error("failed to get asset for deletion",
				zap.Error(err),
				zap.String("asset_id", id.String()),
			)
			return fmt.Errorf("failed to get asset for deletion: %w", err)
		}

		var transactionCount int64
		if err := tx.Model(&entity.Transaction{}).Where("asset_id = ?", id).Count(&transactionCount).Error; err != nil {
			r.logger.Error("failed to check asset transactions",
				zap.Error(err),
				zap.String("asset_id", id.String()),
			)
			return fmt.Errorf("failed to check asset transactions: %w", err)
		}

		if transactionCount > 0 {
			return fmt.Errorf("cannot delete asset: asset is used in %d transactions", transactionCount)
		}

		if err := tx.Delete(&asset).Error; err != nil {
			r.logger.Error("failed to delete asset",
				zap.Error(err),
				zap.String("asset_id", id.String()),
			)
			return fmt.Errorf("failed to delete asset: %w", err)
		}

		cacheKey := cacher.GetAssetCacheKey(id)
		if err := r.redis.Del(ctx, cacheKey); err != nil {
			r.logger.Warn("failed to invalidate asset cache",
				zap.Error(err),
				zap.String("asset_id", id.String()),
			)
		}

		return nil
	})
}
