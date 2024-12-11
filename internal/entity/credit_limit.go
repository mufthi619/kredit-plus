package entity

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type (
	CreditLimit struct {
		ID          uuid.UUID `gorm:"type:char(36);primary_key"`
		CustomerID  uuid.UUID `gorm:"type:char(36);index;not null"`
		TenorMonth  int       `gorm:"type:int;not null"` //In Ex Case : (1, 2, 3, or 6 months)
		LimitAmount float64   `gorm:"type:decimal(15,2);not null"`
		UsedAmount  float64   `gorm:"type:decimal(15,2);not null;default:0"`
		CreatedAt   time.Time `gorm:"type:timestamp;not null"`
		UpdatedAt   time.Time `gorm:"type:timestamp;not null"`
		Customer    Customer  `gorm:"foreignKey:CustomerID"`
	}

	CreditLimitService interface {
		Create(ctx context.Context, req CreateCreditLimitRequest) (*CreditLimitResponse, error)
		GetByID(ctx context.Context, id uuid.UUID) (*CreditLimitResponse, error)
		GetByCustomerIDAndTenor(ctx context.Context, customerID uuid.UUID, tenorMonth int) (*CreditLimitResponse, error)
		GetAllByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CreditLimitResponse, error)
		Delete(ctx context.Context, id uuid.UUID) error
		UpdateUsedAmount(ctx context.Context, id uuid.UUID, amount float64) error
	}

	CreditLimitRepository interface {
		Create(ctx context.Context, limit *CreditLimit) error
		GetByID(ctx context.Context, id uuid.UUID) (*CreditLimit, error)
		GetByCustomerIDAndTenor(ctx context.Context, customerID uuid.UUID, tenorMonth int) (*CreditLimit, error)
		GetAllByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CreditLimit, error)
		UpdateUsedAmount(ctx context.Context, id uuid.UUID, amount float64) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	CreateCreditLimitRequest struct {
		CustomerID  uuid.UUID `json:"customer_id" validate:"required"`
		TenorMonth  int       `json:"tenor_month" validate:"required,oneof=1 2 3 6"`
		LimitAmount float64   `json:"limit_amount" validate:"required,gt=0"`
	}

	CreditLimitResponse struct {
		ID          uuid.UUID `json:"id"`
		CustomerID  uuid.UUID `json:"customer_id"`
		TenorMonth  int       `json:"tenor_month"`
		LimitAmount float64   `json:"limit_amount"`
		UsedAmount  float64   `json:"used_amount"`
		CreatedAt   string    `json:"created_at"`
		UpdatedAt   string    `json:"updated_at"`
	}

	CreditLimitError struct {
		Code    string
		Message string
	}
)

func (r *CreateCreditLimitRequest) Validate() []string {
	var errors []string

	isValidTenor := func(tenor int) bool {
		validTenors := map[int]bool{1: true, 2: true, 3: true, 6: true}
		return validTenors[tenor]
	}

	if r.CustomerID == uuid.Nil {
		errors = append(errors, "customer_id is required")
	}
	if !isValidTenor(r.TenorMonth) {
		errors = append(errors, "tenor_month must be 1, 2, 3, or 6")
	}
	if r.LimitAmount <= 0 {
		errors = append(errors, "limit_amount must be greater than 0")
	}

	return errors
}

func (e *CreditLimitError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
	ErrCreditLimitNotFound     = &CreditLimitError{Code: "CREDIT_LIMIT_NOT_FOUND", Message: "credit limit not found"}
	ErrInsufficientCreditLimit = &CreditLimitError{Code: "INSUFFICIENT_CREDIT_LIMIT", Message: "insufficient credit limit"}
	ErrDuplicateCreditLimit    = &CreditLimitError{Code: "DUPLICATE_CREDIT_LIMIT", Message: "credit limit already exists for this tenor"}
	ErrCreditLimitInUse        = &CreditLimitError{Code: "CREDIT_LIMIT_IN_USE", Message: "credit limit is currently in use"}
)
