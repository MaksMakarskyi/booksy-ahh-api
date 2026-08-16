package hardware

import (
	"cmp"
	"context"
	"fmt"
	"math"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	"github.com/MaksMakarskyi/booksy-go-api/internal/utils/embeddings"
)

func EnsureEmbeddings(ctx context.Context, deps *dependencies.Registry) (int, error) {
	if deps == nil {
		return 0, fmt.Errorf("dependencies registry cannot be nil")
	}
	if deps.Embedder == nil {
		return 0, fmt.Errorf("dependencies registry embedder cannot be nil")
	}

	store, err := NewSQLiteStore(&SQLiteStoreOptions{Client: deps.DB})
	if err != nil {
		return 0, fmt.Errorf("failed to build store: %w", err)
	}

	statuses, err := store.GetAllEmbeddingStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read embedding status: %w", err)
	}

	embedded := 0
	for _, status := range statuses {
		text := status.EmbeddingText()
		if !status.NeedsEmbedding(deps.Embedder.Model(), SourceHash(text)) {
			continue
		}

		if err := embed(ctx, store, deps.Embedder, status.Hardware); err != nil {
			return embedded, fmt.Errorf("failed to embed hardware %d: %w", status.ID, err)
		}

		embedded++
	}

	return embedded, nil
}

func embed(
	ctx context.Context, store Store, embedder embeddings.Embedder, item Hardware,
) error {
	text := item.EmbeddingText()

	vector, err := embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to embed hardware %d: %w", item.ID, err)
	}

	err = store.UpsertEmbedding(ctx, Embedding{
		HardwareID: item.ID,
		Model:      embedder.Model(),
		SourceHash: SourceHash(text),
		Vector:     vector,
	})
	if err != nil {
		return fmt.Errorf("failed to store embedding for hardware %d: %w", item.ID, err)
	}

	return nil
}

type embeddedHardwareWithCosine struct {
	EmbeddedHardware
	CosineSimilarity float32
}

func newEmbeddedHardwareWithCosine(item EmbeddedHardware, queryVector []float32) embeddedHardwareWithCosine {
	return embeddedHardwareWithCosine{
		EmbeddedHardware: item,
		CosineSimilarity: cosineSimilarity(queryVector, item.Vector),
	}
}

// Sorts vectors with higher cosine similarity to the left side and lower cosine
// similarity to the right side. The arguments are reversed because cmp.Compare
// orders ascending and the best match is the highest similarity.
func sortByCosineSimilarity(a, b embeddedHardwareWithCosine) int {
	return cmp.Compare(
		b.CosineSimilarity,
		a.CosineSimilarity,
	)
}

func cosineSimilarity(a, b []float32) float32 {
	return dotProduct(a, b) / (magnitude(a) * magnitude(b))
}

func dotProduct(a, b []float32) float32 {
	product := float32(0)
	for i := range min(len(a), len(b)) {
		product += a[i] * b[i]
	}

	return product
}

func magnitude(v []float32) float32 {
	mag := float64(0)
	for _, val := range v {
		mag += float64(val) * float64(val)
	}

	return float32(math.Sqrt(mag))
}
