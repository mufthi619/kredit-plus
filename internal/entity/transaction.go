package entity

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"html"
	"time"
)

type (
	TransactionStatus       string
	TransactionDetailStatus string

	Transaction struct {
		ID                uuid.UUID          `gorm:"type:char(36);primary_key"`
		CustomerID        uuid.UUID          `gorm:"type:char(36);index;not null"`
		AssetID           uuid.UUID          `gorm:"type:char(36);index;not null"`
		ContractNumber    string             `gorm:"type:varchar(50);unique_index;not null"`
		OTRAmount         float64            `gorm:"type:decimal(15,2);not null"`
		AdminFee          float64            `gorm:"type:decimal(15,2);not null"`
		InterestAmount    float64            `gorm:"type:decimal(15,2);not null"`
		TenorMonth        int                `gorm:"type:int;not null"`
		InstallmentAmount float64            `gorm:"type:decimal(15,2);not null"`
		Status            TransactionStatus  `gorm:"type:varchar(20);not null;check:status in ('pending', 'active', 'completed')"`
		CreatedAt         time.Time          `gorm:"type:timestamp;not null"`
		UpdatedAt         time.Time          `gorm:"type:timestamp;not null"`
		Customer          *Customer          `gorm:"foreignKey:CustomerID"`
		Asset             *Asset             `gorm:"foreignKey:AssetID"`
		TransactionDetail *TransactionDetail `gorm:"foreignKey:TransactionID"`
	}

	TransactionDetail struct {
		ID                uuid.UUID               `gorm:"type:char(36);primary_key"`
		TransactionID     uuid.UUID               `gorm:"type:char(36);index;not null"`
		InstallmentNumber int                     `gorm:"type:int;not null"`
		Amount            float64                 `gorm:"type:decimal(15,2);not null"`
		DueDate           time.Time               `gorm:"type:date;not null"`
		Status            TransactionDetailStatus `gorm:"type:varchar(20);not null;check:status in ('pending', 'paid', 'overdue')"`
		CreatedAt         time.Time               `gorm:"type:timestamp;not null"`
		UpdatedAt         time.Time               `gorm:"type:timestamp;not null"`
	}

	TransactionService interface {
		Create(ctx context.Context, req CreateTransactionRequest) (*TransactionResponse, error)
		GetByID(ctx context.Context, id uuid.UUID) (*TransactionResponse, error)
		GetByContractNumber(ctx context.Context, contractNumber string) (*TransactionResponse, error)
		GetAllByCustomerID(ctx context.Context, customerID uuid.UUID, filter TransactionFilterRequest) ([]TransactionResponse, int64, error)
		UpdateStatus(ctx context.Context, id uuid.UUID, status TransactionStatus) error
	}

	TransactionRepository interface {
		Create(ctx context.Context, transaction *Transaction) error
		GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
		GetByContractNumber(ctx context.Context, contractNumber string) (*Transaction, error)
		GetAllByCustomerID(ctx context.Context, customerID uuid.UUID, filter TransactionFilterRepository) ([]Transaction, int64, error)
		UpdateStatus(ctx context.Context, id uuid.UUID, status TransactionStatus) error
	}

	TransactionFilterRepository struct {
		Status TransactionStatus
		Limit  int
		Offset int
	}

	CreateTransactionRequest struct {
		CustomerID     uuid.UUID `json:"customer_id" validate:"required"`
		AssetID        uuid.UUID `json:"asset_id" validate:"required"`
		TenorMonth     int       `json:"tenor_month" validate:"required,oneof=1 2 3 6"`
		AdminFee       float64   `json:"admin_fee" validate:"required,min=0"`
		InterestRate   float64   `json:"interest_rate" validate:"required,min=0,max=100"`
		ContractNumber string    `json:"contract_number" validate:"required"`
	}

	TransactionFilterRequest struct {
		Status  TransactionStatus `json:"status"`
		Page    int               `json:"page" validate:"min=1"`
		PerPage int               `json:"per_page" validate:"min=1,max=100"`
	}

	TransactionResponse struct {
		ID                uuid.UUID             `json:"id"`
		CustomerID        uuid.UUID             `json:"customer_id"`
		AssetID           uuid.UUID             `json:"asset_id"`
		ContractNumber    string                `json:"contract_number"`
		OTRAmount         float64               `json:"otr_amount"`
		AdminFee          float64               `json:"admin_fee"`
		InterestAmount    float64               `json:"interest_amount"`
		TenorMonth        int                   `json:"tenor_month"`
		InstallmentAmount float64               `json:"installment_amount"`
		Status            TransactionStatus     `json:"status"`
		Asset             AssetResponse         `json:"asset,omitempty"`
		Customer          CustomerResponse      `json:"customer,omitempty"`
		Installments      []InstallmentResponse `json:"installments,omitempty"`
		CreatedAt         string                `json:"created_at"`
		UpdatedAt         string                `json:"updated_at"`
	}

	InstallmentResponse struct {
		ID                uuid.UUID               `json:"id"`
		TransactionID     uuid.UUID               `json:"transaction_id"`
		InstallmentNumber int                     `json:"installment_number"`
		Amount            float64                 `json:"amount"`
		DueDate           string                  `json:"due_date"`
		Status            TransactionDetailStatus `json:"status"`
		CreatedAt         string                  `json:"created_at"`
		UpdatedAt         string                  `json:"updated_at"`
	}

	TransactionError struct {
		Code    string
		Message string
	}
)

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusActive    TransactionStatus = "active"
	TransactionStatusCompleted TransactionStatus = "completed"
)

const (
	TransactionDetailStatusPending TransactionDetailStatus = "pending"
	TransactionDetailStatusPaid    TransactionDetailStatus = "paid"
	TransactionDetailStatusOverdue TransactionDetailStatus = "overdue"
)

func (s TransactionStatus) IsValid() bool {
	switch s {
	case TransactionStatusPending,
		TransactionStatusActive,
		TransactionStatusCompleted:
		return true
	}
	return false
}

func (s TransactionDetailStatus) IsValid() bool {
	switch s {
	case TransactionDetailStatusPending,
		TransactionDetailStatusPaid,
		TransactionDetailStatusOverdue:
		return true
	}
	return false
}

func (r CreateTransactionRequest) Validate() []string {
	var errors []string

	// XSS Protection
	html.EscapeString(r.ContractNumber)

	isValidTenor := func(tenor int) bool {
		validTenors := map[int]bool{1: true, 2: true, 3: true, 6: true}
		return validTenors[tenor]
	}

	if r.CustomerID == uuid.Nil {
		errors = append(errors, "customer_id is required")
	}
	if r.AssetID == uuid.Nil {
		errors = append(errors, "asset_id is required")
	}
	if !isValidTenor(r.TenorMonth) {
		errors = append(errors, "tenor_month must be 1, 2, 3, or 6")
	}
	if r.AdminFee < 0 {
		errors = append(errors, "admin_fee must not be negative")
	}
	if r.InterestRate < 0 || r.InterestRate > 100 {
		errors = append(errors, "interest_rate must be between 0 and 100")
	}
	if r.ContractNumber == "" {
		errors = append(errors, "contract_number is required")
	}

	return errors
}

func (r TransactionFilterRequest) Validate() []string {
	var errors []string

	if r.Page < 1 {
		errors = append(errors, "page must be greater than 0")
	}
	if r.PerPage < 1 {
		errors = append(errors, "per_page must be greater than 0")
	}
	if r.PerPage > 100 {
		errors = append(errors, "per_page must not exceed 100")
	}
	if r.Status != "" && !r.Status.IsValid() {
		errors = append(errors, "invalid status")
	}

	return errors
}

func (r TransactionFilterRequest) ToTransactionFilterRepo() TransactionFilterRepository {
	return TransactionFilterRepository{
		Status: r.Status,
		Limit:  r.PerPage,
		Offset: (r.Page - 1) * r.PerPage,
	}
}

var (
	ErrTransactionNotFound = &TransactionError{Code: "TRANSACTION_NOT_FOUND", Message: "transaction not found"}
	ErrDuplicateContract   = &TransactionError{Code: "DUPLICATE_CONTRACT", Message: "contract number already exists"}
	ErrInvalidStatus       = &TransactionError{Code: "INVALID_STATUS", Message: "invalid transaction status"}
)

func (e *TransactionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
