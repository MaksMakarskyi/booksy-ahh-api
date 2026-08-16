package embeddings

import (
	"context"
	"fmt"
	"math"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/openai/openai-go/v3"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Model() string
	Dimensions() int
}

var _ Embedder = (*OpenAIEmbedder)(nil)

type OpenAIEmbedder struct {
	client     *openai.Client
	model      openai.EmbeddingModel
	dimensions int64
}

type OpenAIEmbedderOptions struct {
	Client     *openai.Client
	Model      openai.EmbeddingModel
	Dimensions int64
}

func NewOpenAIEmbedder(opts *OpenAIEmbedderOptions) (*OpenAIEmbedder, error) {
	if opts == nil {
		return nil, fmt.Errorf("OpenAIEmbedderOptions cannot be nil")
	}
	if opts.Client == nil {
		return nil, fmt.Errorf("OpenAIEmbedderOptions.Client cannot be nil")
	}
	if opts.Model == "" {
		return nil, fmt.Errorf("OpenAIEmbedderOptions.Model cannot be empty")
	}
	if opts.Dimensions <= 0 {
		return nil, fmt.Errorf("OpenAIEmbedderOptions.Dimensions must be greater than 0")
	}

	embedder := &OpenAIEmbedder{
		client:     opts.Client,
		model:      opts.Model,
		dimensions: opts.Dimensions,
	}

	return embedder, nil
}

func (e *OpenAIEmbedder) Model() string { return string(e.model) }

func (e *OpenAIEmbedder) Dimensions() int { return int(e.dimensions) }

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	res, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input:      openai.EmbeddingNewParamsInputUnion{OfString: openai.String(text)},
		Model:      e.model,
		Dimensions: openai.Int(e.dimensions),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrServiceExternal, err)
	}

	if got := len(res.Data); got != 1 {
		return nil, fmt.Errorf(
			"%w: expected exactly one vector, got %d", errutils.ErrServiceInternal, got,
		)
	}

	raw := res.Data[0].Embedding
	if got := len(raw); got != int(e.dimensions) {
		return nil, fmt.Errorf(
			"%w: model returned %d dimensions, expected %d",
			errutils.ErrServiceInternal, got, e.dimensions,
		)
	}

	vector := make([]float32, len(raw))
	for i, value := range raw {
		vector[i] = float32(value)
	}

	normalize(vector)

	return vector, nil
}

// normalize scales a vector to unit length, which makes cosine similarity a
// plain dot product. OpenAI already returns unit-length vectors, so this
// normally changes nothing — it is here so the invariant belongs to this
// codebase rather than to a provider's undocumented behaviour.
func normalize(vector []float32) {
	var sum float64
	for _, value := range vector {
		sum += float64(value) * float64(value)
	}

	norm := math.Sqrt(sum)
	if norm == 0 {
		return
	}

	for i, value := range vector {
		vector[i] = float32(float64(value) / norm)
	}
}
