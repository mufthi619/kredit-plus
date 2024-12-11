package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"strings"
	"sync"
	"time"
)

type transactionService struct {
	transactionRepo entity.TransactionRepository
	customerRepo    entity.CustomerRepository
	creditLimitRepo entity.CreditLimitRepository
	assetRepo       entity.AssetRepository
	logger          *zap.Logger
}

func NewTransactionService(
	transactionRepo entity.TransactionRepository,
	customerRepo entity.CustomerRepository,
	creditLimitRepo entity.CreditLimitRepository,
	assetRepo entity.AssetRepository,
	logger *zap.Logger,
) entity.TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		customerRepo:    customerRepo,
		creditLimitRepo: creditLimitRepo,
		assetRepo:       assetRepo,
		logger:          logger,
	}
}

func (s *transactionService) Create(ctx context.Context, req entity.CreateTransactionRequest) (*entity.TransactionResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	existingTxChan := make(chan struct {
		trx *entity.Transaction
		err error
	})
	customerChan := make(chan struct {
		customer *entity.Customer
		err      error
	})
	assetChan := make(chan struct {
		asset *entity.Asset
		err   error
	})
	creditLimitChan := make(chan struct {
		creditLimit *entity.CreditLimit
		err         error
	})
	wg := &sync.WaitGroup{}
	wg.Add(4)

	go func() {
		defer wg.Done()
		trx, err := s.transactionRepo.GetByContractNumber(ctx, req.ContractNumber)
		existingTxChan <- struct {
			trx *entity.Transaction
			err error
		}{trx, err}
	}()
	go func() {
		defer wg.Done()
		customer, err := s.customerRepo.GetByID(ctx, req.CustomerID)
		customerChan <- struct {
			customer *entity.Customer
			err      error
		}{customer, err}
	}()
	go func() {
		defer wg.Done()
		asset, err := s.assetRepo.GetByID(ctx, req.AssetID)
		assetChan <- struct {
			asset *entity.Asset
			err   error
		}{asset, err}
	}()
	go func() {
		defer wg.Done()
		limit, err := s.creditLimitRepo.GetByCustomerIDAndTenor(ctx, req.CustomerID, req.TenorMonth)
		creditLimitChan <- struct {
			creditLimit *entity.CreditLimit
			err         error
		}{limit, err}
	}()
	go func() {
		wg.Wait()
		close(existingTxChan)
		close(customerChan)
		close(assetChan)
		close(creditLimitChan)
	}()

	existingTxResult := <-existingTxChan
	customerResult := <-customerChan
	assetResult := <-assetChan
	creditLimitResult := <-creditLimitChan

	//Check trx
	if existingTxResult.err != nil {
		s.logger.Error("failed to check existing contract number",
			zap.Error(existingTxResult.err),
			zap.String("contract_number", req.ContractNumber),
		)
		return nil, fmt.Errorf("failed to check existing contract: %w", existingTxResult.err)
	}
	if existingTxResult.trx != nil {
		return nil, entity.ErrDuplicateContract
	}

	//Check Customer
	if customerResult.err != nil {
		s.logger.Error("failed to get customer",
			zap.Error(customerResult.err),
			zap.String("customer_id", req.CustomerID.String()),
		)
		return nil, fmt.Errorf("failed to get customer: %w", customerResult.err)
	}
	if customerResult.customer == nil {
		return nil, fmt.Errorf("customer not found")
	}
	if !customerResult.customer.IsActive {
		return nil, fmt.Errorf("customer is not active")
	}

	//Check Asset
	if assetResult.err != nil {
		s.logger.Error("failed to get asset",
			zap.Error(assetResult.err),
			zap.String("asset_id", req.AssetID.String()),
		)
		return nil, fmt.Errorf("failed to get asset: %w", assetResult.err)
	}
	if assetResult.asset == nil {
		return nil, fmt.Errorf("asset not found")
	}

	//Check Credit Limit
	if creditLimitResult.err != nil {
		s.logger.Error("failed to get credit limit",
			zap.Error(creditLimitResult.err),
			zap.String("customer_id", req.CustomerID.String()),
			zap.Int("tenor_month", req.TenorMonth),
		)
		return nil, fmt.Errorf("failed to get credit limit: %w", creditLimitResult.err)
	}
	if creditLimitResult.creditLimit == nil {
		return nil, fmt.Errorf("no credit limit found for tenor %d months", req.TenorMonth)
	}

	interestAmount := (assetResult.asset.Price * req.InterestRate * float64(req.TenorMonth)) / 100
	totalAmount := assetResult.asset.Price + req.AdminFee + interestAmount
	installmentAmount := totalAmount / float64(req.TenorMonth)

	if totalAmount > creditLimitResult.creditLimit.LimitAmount-creditLimitResult.creditLimit.UsedAmount {
		return nil, entity.ErrInsufficientCreditLimit
	}

	transaction := &entity.Transaction{
		ID:                uuid.New(),
		CustomerID:        req.CustomerID,
		AssetID:           req.AssetID,
		ContractNumber:    req.ContractNumber,
		OTRAmount:         assetResult.asset.Price,
		AdminFee:          req.AdminFee,
		InterestAmount:    interestAmount,
		TenorMonth:        req.TenorMonth,
		InstallmentAmount: installmentAmount,
		Status:            entity.TransactionStatusPending,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		s.logger.Error("failed to create transaction",
			zap.Error(err),
			zap.String("customer_id", req.CustomerID.String()),
		)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	if err := s.creditLimitRepo.UpdateUsedAmount(ctx, creditLimitResult.creditLimit.ID, totalAmount); err != nil {
		s.logger.Error("failed to update credit limit used amount",
			zap.Error(err),
			zap.String("credit_limit_id", creditLimitResult.creditLimit.ID.String()),
		)
		return nil, fmt.Errorf("failed to update credit limit: %w", err)
	}

	createdTx, err := s.transactionRepo.GetByID(ctx, transaction.ID)
	if err != nil {
		s.logger.Error("failed to get created transaction",
			zap.Error(err),
			zap.String("transaction_id", transaction.ID.String()),
		)
		return nil, fmt.Errorf("failed to get created transaction: %w", err)
	}

	return s.toResponse(createdTx), nil
}

func (s *transactionService) GetByID(ctx context.Context, id uuid.UUID) (*entity.TransactionResponse, error) {
	transaction, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get transaction",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if transaction == nil {
		return nil, entity.ErrTransactionNotFound
	}

	return s.toResponse(transaction), nil
}

func (s *transactionService) GetByContractNumber(ctx context.Context, contractNumber string) (*entity.TransactionResponse, error) {
	transaction, err := s.transactionRepo.GetByContractNumber(ctx, contractNumber)
	if err != nil {
		s.logger.Error("failed to get transaction by contract number",
			zap.Error(err),
			zap.String("contract_number", contractNumber),
		)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if transaction == nil {
		return nil, entity.ErrTransactionNotFound
	}

	return s.toResponse(transaction), nil
}

func (s *transactionService) GetAllByCustomerID(ctx context.Context, customerID uuid.UUID, filter entity.TransactionFilterRequest) ([]entity.TransactionResponse, int64, error) {
	if errors := filter.Validate(); len(errors) > 0 {
		return nil, 0, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	transactions, count, err := s.transactionRepo.GetAllByCustomerID(ctx, customerID, filter.ToTransactionFilterRepo())
	if err != nil {
		s.logger.Error("failed to get customer transactions",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	responses := make([]entity.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		responses[i] = *s.toResponse(&tx)
	}

	return responses, count, nil
}

func (s *transactionService) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.TransactionStatus) error {
	if !status.IsValid() {
		return entity.ErrInvalidStatus
	}

	transaction, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get transaction for status update",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
		)
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if transaction == nil {
		return entity.ErrTransactionNotFound
	}

	if err := s.transactionRepo.UpdateStatus(ctx, id, status); err != nil {
		s.logger.Error("failed to update transaction status",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
		)
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (s *transactionService) toResponse(tx *entity.Transaction) *entity.TransactionResponse {
	response := &entity.TransactionResponse{
		ID:                tx.ID,
		CustomerID:        tx.CustomerID,
		AssetID:           tx.AssetID,
		ContractNumber:    tx.ContractNumber,
		OTRAmount:         tx.OTRAmount,
		AdminFee:          tx.AdminFee,
		InterestAmount:    tx.InterestAmount,
		TenorMonth:        tx.TenorMonth,
		InstallmentAmount: tx.InstallmentAmount,
		Status:            tx.Status,
		CreatedAt:         tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         tx.UpdatedAt.Format(time.RFC3339),
	}

	if tx.Asset != nil {
		response.Asset = entity.AssetResponse{
			ID:          tx.Asset.ID,
			Name:        tx.Asset.Name,
			Category:    tx.Asset.Category,
			Description: tx.Asset.Description,
			Price:       tx.Asset.Price,
			CreatedAt:   tx.Asset.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   tx.Asset.UpdatedAt.Format(time.RFC3339),
		}
	}

	if tx.Customer != nil {
		response.Customer = entity.CustomerResponse{
			ID:         tx.Customer.ID,
			NIK:        tx.Customer.NIK,
			FullName:   tx.Customer.FullName,
			LegalName:  tx.Customer.LegalName,
			BirthPlace: tx.Customer.BirthPlace,
			BirthDate:  tx.Customer.BirthDate.Format("2006-01-02"),
			Salary:     tx.Customer.Salary,
			IsActive:   tx.Customer.IsActive,
			CreatedAt:  tx.Customer.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  tx.Customer.UpdatedAt.Format(time.RFC3339),
		}
	}

	if tx.TransactionDetail != nil {
		response.Installments = []entity.InstallmentResponse{
			{
				ID:                tx.TransactionDetail.ID,
				TransactionID:     tx.TransactionDetail.TransactionID,
				InstallmentNumber: tx.TransactionDetail.InstallmentNumber,
				Amount:            tx.TransactionDetail.Amount,
				DueDate:           tx.TransactionDetail.DueDate.Format("2006-01-02"),
				Status:            tx.TransactionDetail.Status,
				CreatedAt:         tx.TransactionDetail.CreatedAt.Format(time.RFC3339),
				UpdatedAt:         tx.TransactionDetail.UpdatedAt.Format(time.RFC3339),
			},
		}
	}

	return response
}
