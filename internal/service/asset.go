package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"strings"
	"time"
)

type assetService struct {
	repo   entity.AssetRepository
	logger *zap.Logger
}

func NewAssetService(repo entity.AssetRepository, logger *zap.Logger) entity.AssetService {
	return &assetService{
		repo:   repo,
		logger: logger,
	}
}

func (s *assetService) Create(ctx context.Context, req entity.CreateAssetRequest) (*entity.AssetResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	asset := &entity.Asset{
		ID:          uuid.New(),
		Name:        req.Name,
		Category:    req.Category,
		Description: req.Description,
		Price:       req.Price,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, asset); err != nil {
		s.logger.Error("failed to create asset", zap.Error(err))
		return nil, err
	}

	return s.toResponse(asset), nil
}

func (s *assetService) GetByID(ctx context.Context, id uuid.UUID) (*entity.AssetResponse, error) {
	asset, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get asset", zap.Error(err))
		return nil, err
	}

	if asset == nil {
		return nil, fmt.Errorf("asset not found")
	}

	return s.toResponse(asset), nil
}

func (s *assetService) GetAll(ctx context.Context, filter entity.AssetFilterRequest) ([]entity.AssetResponse, int64, error) {
	assets, count, err := s.repo.GetAllWithFilter(ctx, filter.ToAssetFilterRepo())
	if err != nil {
		s.logger.Error("failed to get assets", zap.Error(err))
		return nil, 0, err
	}

	responses := make([]entity.AssetResponse, len(assets))
	for i, asset := range assets {
		responses[i] = *s.toResponse(&asset)
	}

	return responses, count, nil
}

func (s *assetService) Update(ctx context.Context, id uuid.UUID, req entity.UpdateAssetRequest) (*entity.AssetResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errors)
	}

	asset, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get asset for update", zap.Error(err))
		return nil, err
	}

	if asset == nil {
		return nil, fmt.Errorf("asset not found")
	}

	asset.Name = req.Name
	asset.Description = req.Description
	asset.Price = req.Price
	asset.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, asset); err != nil {
		s.logger.Error("failed to update asset", zap.Error(err))
		return nil, err
	}

	return s.toResponse(asset), nil
}

func (s *assetService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete asset", zap.Error(err))
		return err
	}
	return nil
}

func (s *assetService) toResponse(asset *entity.Asset) *entity.AssetResponse {
	return &entity.AssetResponse{
		ID:          asset.ID,
		Name:        asset.Name,
		Category:    asset.Category,
		Description: asset.Description,
		Price:       asset.Price,
		CreatedAt:   asset.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   asset.UpdatedAt.Format(time.RFC3339),
	}
}
