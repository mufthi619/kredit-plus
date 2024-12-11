package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"kredit-plus/utils/response_formatter"
	"strconv"
)

type AssetHandler struct {
	service entity.AssetService
	logger  *zap.Logger
}

func NewAssetHandler(service entity.AssetService, logger *zap.Logger) *AssetHandler {
	return &AssetHandler{
		service: service,
		logger:  logger,
	}
}

func (h *AssetHandler) RegisterRoutes(app *fiber.App) {
	assets := app.Group("/api/v1/assets")
	assets.Post("", h.Create)
	assets.Get("", h.List)
	assets.Get("/:id", h.GetByID)
	assets.Put("/:id", h.Update)
	assets.Delete("/:id", h.Delete)
}

func (h *AssetHandler) Create(c *fiber.Ctx) error {
	var req entity.CreateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	asset, err := h.service.Create(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to create asset",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusCreated).JSON(response_formatter.Created(asset, "Asset created successfully"))
}

func (h *AssetHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))
	page, perPage = response_formatter.ValidatePagination(page, perPage)

	filter := entity.AssetFilterRequest{
		Category: c.Query("category"),
		MinPrice: func() float64 {
			v, _ := strconv.ParseFloat(c.Query("min_price"), 64)
			return v
		}(),
		MaxPrice: func() float64 {
			v, _ := strconv.ParseFloat(c.Query("max_price"), 64)
			return v
		}(),
		Limit:  perPage,
		Offset: response_formatter.CalculateOffset(page, perPage),
	}

	assets, total, err := h.service.GetAll(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to fetch assets",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.WithPagination(
		assets,
		"Assets retrieved successfully",
		page,
		perPage,
		total,
	))
}

func (h *AssetHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid asset ID",
			[]string{err.Error()},
		))
	}

	asset, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
			fiber.StatusNotFound,
			"Asset not found",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(asset, "Asset retrieved successfully"))
}

func (h *AssetHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid asset ID",
			[]string{err.Error()},
		))
	}

	var req entity.UpdateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	asset, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to update asset",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(asset, "Asset updated successfully"))
}

func (h *AssetHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid asset ID",
			[]string{err.Error()},
		))
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to delete asset",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(nil, "Asset deleted successfully"))
}
