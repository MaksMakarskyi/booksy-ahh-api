package hardware

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/MaksMakarskyi/booksy-go-api/internal/utils/embeddings"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
	"github.com/labstack/echo/v5"
)

type Handler struct {
	store      Store
	embedder   embeddings.Embedder
	searchTopK int
}

type HandlerOptions struct {
	Store      Store
	Embedder   embeddings.Embedder
	SearchTopK int
}

func NewHandler(opts *HandlerOptions) (*Handler, error) {
	if opts == nil {
		return nil, fmt.Errorf("HandlerOptions cannot be nil")
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("HandlerOptions.Store cannot be nil")
	}
	if opts.Embedder == nil {
		return nil, fmt.Errorf("HandlerOptions.Embedder cannot be nil")
	}
	if opts.SearchTopK <= 0 {
		return nil, fmt.Errorf("HandlerOptions.SearchTopK must be greater than 0")
	}

	handler := &Handler{
		store:      opts.Store,
		embedder:   opts.Embedder,
		searchTopK: opts.SearchTopK,
	}

	return handler, nil
}

func (h *Handler) GetAll(c *echo.Context) error {
	items, err := h.store.GetAll(c.Request().Context())
	if err != nil {
		return fmt.Errorf("failed to get all hardware items: %w", err)
	}

	return c.JSON(http.StatusOK, map[string][]Hardware{
		"data": items,
	})
}

func (h *Handler) Create(c *echo.Context) error {
	var newHardware NewHardware
	if err := valutils.DecodeJSON(c, &newHardware); err != nil {
		return err
	}

	storedHardware, err := h.store.Create(c.Request().Context(), newHardware)
	if err != nil {
		return fmt.Errorf("failed to store hardware: %w", err)
	}

	h.refreshEmbedding(c, storedHardware)

	return c.JSON(http.StatusCreated, map[string]Hardware{
		"data": storedHardware,
	})
}

func (h *Handler) Update(c *echo.Context) error {
	var updatedHardware UpdatedHardware
	if err := valutils.DecodeJSON(c, &updatedHardware); err != nil {
		return err
	}

	storedHardware, err := h.store.Update(c.Request().Context(), updatedHardware)
	if err != nil {
		return fmt.Errorf("failed to update hardware %d: %w", updatedHardware.ID, err)
	}

	h.refreshEmbedding(c, storedHardware)

	return c.JSON(http.StatusOK, map[string]Hardware{
		"data": storedHardware,
	})
}

func (h *Handler) Delete(c *echo.Context) error {
	hardwareID, err := valutils.PathInt(c, "id")
	if err != nil {
		return err
	}

	deletedHardware, err := h.store.Delete(c.Request().Context(), hardwareID)
	if err != nil {
		return fmt.Errorf("failed to delete hardware %d: %w", hardwareID, err)
	}

	return c.JSON(http.StatusOK, map[string]Hardware{
		"data": deletedHardware,
	})
}

func (h *Handler) MarkRepair(c *echo.Context) error {
	hardwareID, err := valutils.PathInt(c, "id")
	if err != nil {
		return err
	}

	markedRepairHardware, err := h.store.MarkRepair(c.Request().Context(), hardwareID)
	if err != nil {
		return fmt.Errorf("failed to mark repair hardware %d: %w", hardwareID, err)
	}

	return c.JSON(http.StatusOK, map[string]Hardware{
		"data": markedRepairHardware,
	})
}

func (h *Handler) MarkAvailable(c *echo.Context) error {
	hardwareID, err := valutils.PathInt(c, "id")
	if err != nil {
		return err
	}

	markedAvailableHardware, err := h.store.MarkAvailable(c.Request().Context(), hardwareID)
	if err != nil {
		return fmt.Errorf("failed to mark available hardware %d: %w", hardwareID, err)
	}

	return c.JSON(http.StatusOK, map[string]Hardware{
		"data": markedAvailableHardware,
	})
}

func (h *Handler) Search(c *echo.Context) error {
	var req SearchRequest
	if err := valutils.DecodeJSON(c, &req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	queryVector, err := h.embedder.Embed(ctx, req.Query)
	if err != nil {
		return fmt.Errorf("failed to embed search query: %w", err)
	}

	candidates, err := h.store.GetAllEmbeddings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get hardware embeddings: %w", err)
	}

	comparableWithCosine := make([]embeddedHardwareWithCosine, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Model == h.embedder.Model() && len(candidate.Vector) == len(queryVector) {
			comparableWithCosine = append(comparableWithCosine,
				newEmbeddedHardwareWithCosine(candidate, queryVector))
		}
	}

	slices.SortFunc(comparableWithCosine, sortByCosineSimilarity)

	items := make([]Hardware, 0, h.searchTopK)
	for _, match := range comparableWithCosine[:min(len(comparableWithCosine), h.searchTopK)] {
		items = append(items, match.Hardware)
	}

	return c.JSON(http.StatusOK, map[string][]Hardware{
		"data": items,
	})
}

func (h *Handler) refreshEmbedding(c *echo.Context, item Hardware) {
	ctx := c.Request().Context()

	status, err := h.store.GetEmbeddingStatus(ctx, item.ID)
	if err != nil {
		c.Logger().Error("failed to read embedding status", "hardware_id", item.ID, "error", err)

		return
	}

	text := item.EmbeddingText()
	if !status.NeedsEmbedding(h.embedder.Model(), SourceHash(text)) {
		return
	}

	if err := embed(ctx, h.store, h.embedder, item); err != nil {
		c.Logger().Error("failed to embed hardware", "hardware_id", item.ID, "error", err)
	}
}
