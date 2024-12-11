package entity

import (
	"context"
	"github.com/google/uuid"
	"time"
)

type (
	DocumentType string

	Customer struct {
		ID           uuid.UUID          `gorm:"type:char(36);primary_key"`
		NIK          string             `gorm:"type:varchar(16);unique_index;not null"`
		FullName     string             `gorm:"type:varchar(100);not null"`
		LegalName    string             `gorm:"type:varchar(100);not null"`
		BirthPlace   string             `gorm:"type:varchar(100);not null"`
		BirthDate    time.Time          `gorm:"type:date;not null"`
		Salary       float64            `gorm:"type:decimal(15,2);not null"`
		IsActive     bool               `gorm:"type:boolean;default:true"`
		CreatedAt    time.Time          `gorm:"type:timestamp;not null"`
		UpdatedAt    time.Time          `gorm:"type:timestamp;not null"`
		Documents    []CustomerDocument `gorm:"foreignKey:CustomerID"`
		CreditLimits []CreditLimit      `gorm:"foreignKey:CustomerID"`
		Transactions []Transaction      `gorm:"foreignKey:CustomerID"`
	}

	CustomerDocument struct {
		ID           uuid.UUID    `gorm:"type:char(36);primary_key"`
		CustomerID   uuid.UUID    `gorm:"type:char(36);index;not null"`
		DocumentType DocumentType `gorm:"type:varchar(50);not null;check:document_type in ('ktp', 'selfie')"`
		DocumentURL  string       `gorm:"type:varchar(255);not null"`
		CreatedAt    time.Time    `gorm:"type:timestamp;not null"`
		UpdatedAt    time.Time    `gorm:"type:timestamp;not null"`
		Customer     Customer     `gorm:"foreignKey:CustomerID"`
	}

	CustomerService interface {
		Create(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error)
		GetByID(ctx context.Context, id uuid.UUID) (*CustomerResponse, error)
		GetByNIK(ctx context.Context, nik string) (*CustomerResponse, error)
		Update(ctx context.Context, id uuid.UUID, req UpdateCustomerRequest) (*CustomerResponse, error)
		Delete(ctx context.Context, id uuid.UUID) error
		UploadDocument(ctx context.Context, customerID uuid.UUID, req UploadDocumentRequest) (*CustomerDocumentResponse, error)
		GetDocuments(ctx context.Context, customerID uuid.UUID, filter DocumentFilterRequest) ([]CustomerDocumentResponse, int64, error)
	}

	CustomerRepository interface {
		Create(ctx context.Context, customer *Customer) error
		GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
		GetByNIK(ctx context.Context, nik string) (*Customer, error)
		Update(ctx context.Context, customer *Customer) error
		Delete(ctx context.Context, id uuid.UUID) error
		CreateDocument(ctx context.Context, doc *CustomerDocument) error
		GetDocuments(ctx context.Context, filter DocumentFilterRepository) (documents []CustomerDocument, count int64, err error)
	}

	DocumentFilterRepository struct {
		CustomerID   uuid.UUID
		DocumentType *DocumentType
		Limit        int
		Offset       int
	}

	CreateCustomerRequest struct {
		NIK        string    `json:"nik" validate:"required,len=16"`
		FullName   string    `json:"full_name" validate:"required,max=100"`
		LegalName  string    `json:"legal_name" validate:"required,max=100"`
		BirthPlace string    `json:"birth_place" validate:"required"`
		BirthDate  time.Time `json:"birth_date" validate:"required"`
		Salary     float64   `json:"salary" validate:"required,min=0"`
	}

	UpdateCustomerRequest struct {
		FullName   string    `json:"full_name" validate:"required,max=100"`
		LegalName  string    `json:"legal_name" validate:"required,max=100"`
		BirthPlace string    `json:"birth_place" validate:"required"`
		BirthDate  time.Time `json:"birth_date" validate:"required"`
		Salary     float64   `json:"salary" validate:"required,min=0"`
	}

	UploadDocumentRequest struct {
		DocumentType DocumentType `json:"document_type" validate:"required,oneof=ktp selfie"`
		DocumentURL  string       `json:"document_url" validate:"required,url"`
	}

	DocumentFilterRequest struct {
		DocumentType *DocumentType `json:"document_type"`
		Page         int           `json:"page" validate:"min=1"`
		PerPage      int           `json:"per_page" validate:"min=1,max=100"`
	}

	CustomerResponse struct {
		ID         uuid.UUID                  `json:"id"`
		NIK        string                     `json:"nik"`
		FullName   string                     `json:"full_name"`
		LegalName  string                     `json:"legal_name"`
		BirthPlace string                     `json:"birth_place"`
		BirthDate  string                     `json:"birth_date"` // Format: YYYY-MM-DD
		Salary     float64                    `json:"salary"`
		IsActive   bool                       `json:"is_active"`
		Documents  []CustomerDocumentResponse `json:"documents,omitempty"`
		CreatedAt  string                     `json:"created_at"` // RFC3339 format
		UpdatedAt  string                     `json:"updated_at"` // RFC3339 format
	}

	CustomerDocumentResponse struct {
		ID           uuid.UUID    `json:"id"`
		CustomerID   uuid.UUID    `json:"customer_id"`
		DocumentType DocumentType `json:"document_type"`
		DocumentURL  string       `json:"document_url"`
		CreatedAt    string       `json:"created_at"` // RFC3339 format
		UpdatedAt    string       `json:"updated_at"` // RFC3339 format
	}

	DocumentListResponse struct {
		Documents []CustomerDocumentResponse `json:"documents"`
		Metadata  PaginationMetadata         `json:"metadata"`
	}

	PaginationMetadata struct {
		CurrentPage int   `json:"current_page"`
		PerPage     int   `json:"per_page"`
		TotalPages  int   `json:"total_pages"`
		TotalItems  int64 `json:"total_items"`
	}
)

const (
	DocumentTypeKTP    DocumentType = "ktp"
	DocumentTypeSelfie DocumentType = "selfie"
)

func (dt DocumentType) IsValid() bool {
	switch dt {
	case DocumentTypeKTP,
		DocumentTypeSelfie:
		return true
	}
	return false
}

func (r CreateCustomerRequest) Validate() []string {
	var errors []string
	if len(r.NIK) != 16 {
		errors = append(errors, "NIK must be 16 characters")
	}
	if r.FullName == "" {
		errors = append(errors, "full name is required")
	}
	if len(r.FullName) > 100 {
		errors = append(errors, "full name must not exceed 100 characters")
	}
	if r.LegalName == "" {
		errors = append(errors, "legal name is required")
	}
	if len(r.LegalName) > 100 {
		errors = append(errors, "legal name must not exceed 100 characters")
	}
	if r.BirthPlace == "" {
		errors = append(errors, "birth place is required")
	}
	if r.BirthDate.IsZero() {
		errors = append(errors, "birth date is required")
	}
	if r.Salary <= 0 {
		errors = append(errors, "salary must be greater than 0")
	}
	return errors
}

func (r UpdateCustomerRequest) Validate() []string {
	var errors []string
	if r.FullName == "" {
		errors = append(errors, "full name is required")
	}
	if len(r.FullName) > 100 {
		errors = append(errors, "full name must not exceed 100 characters")
	}
	if r.LegalName == "" {
		errors = append(errors, "legal name is required")
	}
	if len(r.LegalName) > 100 {
		errors = append(errors, "legal name must not exceed 100 characters")
	}
	if r.BirthPlace == "" {
		errors = append(errors, "birth place is required")
	}
	if r.BirthDate.IsZero() {
		errors = append(errors, "birth date is required")
	}
	if r.Salary <= 0 {
		errors = append(errors, "salary must be greater than 0")
	}
	return errors
}

func (r UploadDocumentRequest) Validate() []string {
	var errors []string
	if !r.DocumentType.IsValid() {
		errors = append(errors, "invalid document type, must be either 'ktp' or 'selfie'")
	}
	if r.DocumentURL == "" {
		errors = append(errors, "document URL is required")
	}
	if len(r.DocumentURL) < 10 || len(r.DocumentURL) > 255 {
		errors = append(errors, "document URL must be between 10 and 255 characters")
	}
	return errors
}

func (r DocumentFilterRequest) Validate() []string {
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
	if r.DocumentType != nil && !r.DocumentType.IsValid() {
		errors = append(errors, "invalid document type")
	}
	return errors
}

func (r DocumentFilterRequest) ToDocumentFilterRepo(customerID uuid.UUID) DocumentFilterRepository {
	return DocumentFilterRepository{
		CustomerID:   customerID,
		DocumentType: r.DocumentType,
		Limit:        r.PerPage,
		Offset:       (r.Page - 1) * r.PerPage,
	}
}
