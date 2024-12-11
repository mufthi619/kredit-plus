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

type creditLimitService struct {
	repo   entity.CreditLimitRepository
	logger *zap.Logger
}

func NewCreditLimitService(repo entity.CreditLimitRepository, logger *zap.Logger) entity.CreditLimitService {
	return &creditLimitService{
		repo:   repo,
		logger: logger,
	}
}

func (s *creditLimitService) Create(ctx context.Context, req entity.CreateCreditLimitRequest) (*entity.CreditLimitResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	existingLimit, err := s.repo.GetByCustomerIDAndTenor(ctx, req.CustomerID, req.TenorMonth)
	if err != nil {
		s.logger.Error("failed to check existing credit limit",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID.String()),
			zap.Int("tenor_month", req.TenorMonth),
		)
		return nil, fmt.Errorf("failed to check existing credit limit: %w", err)
	}

	if existingLimit != nil {
		return nil, entity.ErrDuplicateCreditLimit
	}

	limit := &entity.CreditLimit{
		ID:          uuid.New(),
		CustomerID:  req.CustomerID,
		TenorMonth:  req.TenorMonth,
		LimitAmount: req.LimitAmount,
		UsedAmount:  0,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, limit); err != nil {
		s.logger.Error("failed to create credit limit",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID.String()),
			zap.Int("tenor_month", req.TenorMonth),
		)
		return nil, fmt.Errorf("failed to create credit limit: %w", err)
	}

	return s.toResponse(limit), nil
}

func (s *creditLimitService) GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditLimitResponse, error) {
	limit, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get credit limit by ID",
			zap.Error(err),
			zap.String("credit_limit_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get credit limit: %w", err)
	}

	if limit == nil {
		return nil, entity.ErrCreditLimitNotFound
	}

	return s.toResponse(limit), nil
}

func (s *creditLimitService) GetByCustomerIDAndTenor(ctx context.Context, customerID uuid.UUID, tenorMonth int) (*entity.CreditLimitResponse, error) {
	isValidTenor := func(tenor int) bool {
		validTenors := map[int]bool{1: true, 2: true, 3: true, 6: true}
		return validTenors[tenor]
	}

	if !isValidTenor(tenorMonth) {
		return nil, fmt.Errorf("invalid tenor month: must be 1, 2, 3, or 6")
	}

	limit, err := s.repo.GetByCustomerIDAndTenor(ctx, customerID, tenorMonth)
	if err != nil {
		s.logger.Error("failed to get credit limit by customer ID and tenor",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
			zap.Int("tenor_month", tenorMonth),
		)
		return nil, fmt.Errorf("failed to get credit limit: %w", err)
	}

	if limit == nil {
		return nil, entity.ErrCreditLimitNotFound
	}

	return s.toResponse(limit), nil
}

func (s *creditLimitService) GetAllByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CreditLimitResponse, error) {
	limits, err := s.repo.GetAllByCustomerID(ctx, customerID)
	if err != nil {
		s.logger.Error("failed to get credit limits by customer ID",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, fmt.Errorf("failed to get credit limits: %w", err)
	}

	responses := make([]entity.CreditLimitResponse, len(limits))
	for i, limit := range limits {
		responses[i] = *s.toResponse(&limit)
	}

	return responses, nil
}

func (s *creditLimitService) Delete(ctx context.Context, id uuid.UUID) error {
	limit, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get credit limit for deletion",
			zap.Error(err),
			zap.String("credit_limit_id", id.String()),
		)
		return fmt.Errorf("failed to get credit limit: %w", err)
	}

	if limit == nil {
		return entity.ErrCreditLimitNotFound
	}

	if limit.UsedAmount > 0 {
		return entity.ErrCreditLimitInUse
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete credit limit",
			zap.Error(err),
			zap.String("credit_limit_id", id.String()),
		)
		return fmt.Errorf("failed to delete credit limit: %w", err)
	}

	return nil
}

func (s *creditLimitService) UpdateUsedAmount(ctx context.Context, id uuid.UUID, amount float64) error {
	if amount == 0 {
		return nil
	}

	limit, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get credit limit for updating used amount",
			zap.Error(err),
			zap.String("credit_limit_id", id.String()),
		)
		return fmt.Errorf("failed to get credit limit: %w", err)
	}

	if limit == nil {
		return entity.ErrCreditLimitNotFound
	}

	if amount < 0 {
		if limit.UsedAmount+amount < 0 {
			return fmt.Errorf("invalid amount: would result in negative used amount")
		}
	}
	
	if amount > 0 {
		if limit.UsedAmount+amount > limit.LimitAmount {
			return entity.ErrInsufficientCreditLimit
		}
	}

	if err := s.repo.UpdateUsedAmount(ctx, id, amount); err != nil {
		s.logger.Error("failed to update credit limit used amount",
			zap.Error(err),
			zap.String("credit_limit_id", id.String()),
			zap.Float64("amount", amount),
		)
		return fmt.Errorf("failed to update credit limit used amount: %w", err)
	}

	return nil
}

func (s *creditLimitService) toResponse(limit *entity.CreditLimit) *entity.CreditLimitResponse {
	return &entity.CreditLimitResponse{
		ID:          limit.ID,
		CustomerID:  limit.CustomerID,
		TenorMonth:  limit.TenorMonth,
		LimitAmount: limit.LimitAmount,
		UsedAmount:  limit.UsedAmount,
		CreatedAt:   limit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   limit.UpdatedAt.Format(time.RFC3339),
	}
}
