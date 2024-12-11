package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"kredit-plus/utils/response_formatter"
	"strconv"
)

type CreditLimitHandler struct {
	service entity.CreditLimitService
	logger  *zap.Logger
}

func NewCreditLimitHandler(service entity.CreditLimitService, logger *zap.Logger) *CreditLimitHandler {
	return &CreditLimitHandler{
		service: service,
		logger:  logger,
	}
}

func (h *CreditLimitHandler) RegisterRoutes(app *fiber.App) {
	creditLimits := app.Group("/api/v1/credit-limits")
	creditLimits.Post("", h.Create)
	creditLimits.Get("/:id", h.GetByID)
	creditLimits.Get("/customer/:customer_id", h.GetAllByCustomerID)
	creditLimits.Get("/customer/:customer_id/tenor/:tenor_month", h.GetByCustomerIDAndTenor)
	creditLimits.Put("/:id/used-amount", h.UpdateUsedAmount)
	creditLimits.Delete("/:id", h.Delete)
}

func (h *CreditLimitHandler) Create(c *fiber.Ctx) error {
	var req entity.CreateCreditLimitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	creditLimit, err := h.service.Create(c.Context(), req)
	if err != nil {
		if err == entity.ErrDuplicateCreditLimit {
			return c.Status(fiber.StatusConflict).JSON(response_formatter.Error(
				fiber.StatusConflict,
				"Credit limit already exists",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to create credit limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to create credit limit",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusCreated).JSON(response_formatter.Created(
		creditLimit,
		"Credit limit created successfully",
	))
}

func (h *CreditLimitHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid credit limit ID",
			[]string{err.Error()},
		))
	}

	creditLimit, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == entity.ErrCreditLimitNotFound {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Credit limit not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get credit limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get credit limit",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		creditLimit,
		"Credit limit retrieved successfully",
	))
}

func (h *CreditLimitHandler) GetAllByCustomerID(c *fiber.Ctx) error {
	customerID, err := uuid.Parse(c.Params("customer_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	creditLimits, err := h.service.GetAllByCustomerID(c.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to get customer credit limits", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get credit limits",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		creditLimits,
		"Credit limits retrieved successfully",
	))
}

func (h *CreditLimitHandler) GetByCustomerIDAndTenor(c *fiber.Ctx) error {
	customerID, err := uuid.Parse(c.Params("customer_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	tenorMonth, err := strconv.Atoi(c.Params("tenor_month"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid tenor month",
			[]string{err.Error()},
		))
	}

	creditLimit, err := h.service.GetByCustomerIDAndTenor(c.Context(), customerID, tenorMonth)
	if err != nil {
		if err == entity.ErrCreditLimitNotFound {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Credit limit not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get credit limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get credit limit",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		creditLimit,
		"Credit limit retrieved successfully",
	))
}

type UpdateUsedAmountRequest struct {
	Amount float64 `json:"amount" validate:"required"`
}

func (h *CreditLimitHandler) UpdateUsedAmount(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid credit limit ID",
			[]string{err.Error()},
		))
	}

	var req UpdateUsedAmountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	if err := h.service.UpdateUsedAmount(c.Context(), id, req.Amount); err != nil {
		if err == entity.ErrCreditLimitNotFound {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Credit limit not found",
				[]string{err.Error()},
			))
		}

		if err == entity.ErrInsufficientCreditLimit {
			return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
				fiber.StatusBadRequest,
				"Insufficient credit limit",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to update credit limit used amount", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to update credit limit used amount",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		nil,
		"Credit limit used amount updated successfully",
	))
}

func (h *CreditLimitHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid credit limit ID",
			[]string{err.Error()},
		))
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err == entity.ErrCreditLimitNotFound {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Credit limit not found",
				[]string{err.Error()},
			))
		}

		if err == entity.ErrCreditLimitInUse {
			return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
				fiber.StatusBadRequest,
				"Cannot delete credit limit in use",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to delete credit limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to delete credit limit",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		nil,
		"Credit limit deleted successfully",
	))
}
