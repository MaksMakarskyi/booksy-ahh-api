package hardware

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	errutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/errors"
	"github.com/georgysavva/scany/v2/sqlscan"
)

const (
	hardwareColumns = "id, name, brand, description, purchase_date, status, created_at, updated_at"

	getAllQuery = `
SELECT ` + hardwareColumns + `
FROM hardware`

	createQuery = `
INSERT INTO hardware (name, brand, description, purchase_date)
VALUES ($1, $2, $3, $4)
RETURNING ` + hardwareColumns

	updateQuery = `
UPDATE hardware
SET name          = COALESCE($1, name),
    brand         = COALESCE($2, brand),
    description   = COALESCE($3, description),
    purchase_date = COALESCE($4, purchase_date)
WHERE id = $5
RETURNING ` + hardwareColumns

	deleteQuery = `
DELETE FROM hardware
WHERE id = $1
RETURNING ` + hardwareColumns

	selectHardwareStatusQuery = `
SELECT status
FROM hardware
WHERE id = $1`

	markRepairQuery = `
UPDATE hardware
SET status = 'repair'
WHERE id = $1 AND status = 'available'
RETURNING ` + hardwareColumns

	markAvailableQuery = `
UPDATE hardware
SET status = 'available'
WHERE id = $1 AND status = 'repair'
RETURNING ` + hardwareColumns

	joinedHardwareColumns = `h.id, h.name, h.brand, h.description,
       h.purchase_date, h.status, h.created_at, h.updated_at`

	upsertEmbeddingQuery = `
INSERT INTO hardware_embeddings (hardware_id, model, dimensions, source_hash, vector)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (hardware_id) DO UPDATE
SET model       = excluded.model,
    dimensions  = excluded.dimensions,
    source_hash = excluded.source_hash,
    vector      = excluded.vector`

	getAllEmbeddingsQuery = `
SELECT ` + joinedHardwareColumns + `,
       e.model  AS embedding_model,
       e.vector AS vector
FROM hardware h
JOIN hardware_embeddings e ON e.hardware_id = h.id`

	embeddingStatusQuery = `
SELECT ` + joinedHardwareColumns + `,
       e.model       AS embedding_model,
       e.source_hash AS embedding_source_hash
FROM hardware h
LEFT JOIN hardware_embeddings e ON e.hardware_id = h.id`

	getEmbeddingStatusQuery = embeddingStatusQuery + `
WHERE h.id = $1`

	getAllEmbeddingStatusQuery = embeddingStatusQuery
)

type Store interface {
	GetAll(ctx context.Context) ([]Hardware, error)
	Create(ctx context.Context, newHardware NewHardware) (Hardware, error)
	Update(ctx context.Context, updatedHardware UpdatedHardware) (Hardware, error)
	Delete(ctx context.Context, hardwareID int) (Hardware, error)

	MarkRepair(ctx context.Context, hardwareID int) (Hardware, error)
	MarkAvailable(ctx context.Context, hardwareID int) (Hardware, error)

	UpsertEmbedding(ctx context.Context, embedding Embedding) error
	GetAllEmbeddings(ctx context.Context) ([]EmbeddedHardware, error)
	GetEmbeddingStatus(ctx context.Context, hardwareID int) (EmbeddingStatus, error)
	GetAllEmbeddingStatus(ctx context.Context) ([]EmbeddingStatus, error)
}

var _ Store = (*SQLiteStore)(nil)

type SQLiteStore struct {
	client *sql.DB
}

type SQLiteStoreOptions struct {
	Client *sql.DB
}

func NewSQLiteStore(opts *SQLiteStoreOptions) (*SQLiteStore, error) {
	if opts == nil {
		return nil, fmt.Errorf("SQLiteStoreOptions cannot be nil")
	}
	if opts.Client == nil {
		return nil, fmt.Errorf("SQLiteStoreOptions.Client cannot be nil")
	}

	return &SQLiteStore{client: opts.Client}, nil
}

func (s *SQLiteStore) GetAll(ctx context.Context) ([]Hardware, error) {
	res := make([]Hardware, 0)
	if err := sqlscan.Select(ctx, s.client, &res, getAllQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Create(ctx context.Context, newHardware NewHardware) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, s.client, &res, createQuery,
		newHardware.Name,
		newHardware.Brand,
		newHardware.Description,
		newHardware.PurchaseDate,
	); err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Update(ctx context.Context, updatedHardware UpdatedHardware) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, s.client, &res, updateQuery,
		updatedHardware.Name,
		updatedHardware.Brand,
		updatedHardware.Description,
		updatedHardware.PurchaseDate,
		updatedHardware.ID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", updatedHardware.ID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, hardwareID int) (Hardware, error) {
	var res Hardware
	if err := sqlscan.Get(ctx, s.client, &res, deleteQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) MarkRepair(ctx context.Context, hardwareID int) (Hardware, error) {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxBegin, err)
	}
	defer tx.Rollback()

	var status string
	if err := sqlscan.Get(ctx, tx, &status, selectHardwareStatusQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	hardwareStatus := HardwareStatus(status)
	if !hardwareStatus.IsValid() {
		return Hardware{}, fmt.Errorf(
			"%w: hardware %d has invalid status: %q",
			errutils.ErrStoreInternal, hardwareID, hardwareStatus,
		)
	}
	if hardwareStatus != Available {
		return Hardware{}, fmt.Errorf(
			"%w: hardware %d is %s", errutils.ErrStoreConflict, hardwareID, status,
		)
	}

	var res Hardware
	if err := sqlscan.Get(ctx, tx, &res, markRepairQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	if err := tx.Commit(); err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxCommit, err)
	}

	return res, nil
}

func (s *SQLiteStore) MarkAvailable(ctx context.Context, hardwareID int) (Hardware, error) {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxBegin, err)
	}
	defer tx.Rollback()

	var status string
	if err := sqlscan.Get(ctx, tx, &status, selectHardwareStatusQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	hardwareStatus := HardwareStatus(status)
	if !hardwareStatus.IsValid() {
		return Hardware{}, fmt.Errorf(
			"%w: hardware %d has invalid status: %q",
			errutils.ErrStoreInternal, hardwareID, hardwareStatus,
		)
	}
	if hardwareStatus != Repair {
		return Hardware{}, fmt.Errorf(
			"%w: hardware %d is %s", errutils.ErrStoreConflict, hardwareID, status,
		)
	}

	var res Hardware
	if err := sqlscan.Get(ctx, tx, &res, markAvailableQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Hardware{}, fmt.Errorf("hardware %d: %w", hardwareID, errutils.ErrStoreNotFound)
		}

		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	if err := tx.Commit(); err != nil {
		return Hardware{}, fmt.Errorf("%w: %w", errutils.ErrStoreTxCommit, err)
	}

	return res, nil
}

func (s *SQLiteStore) UpsertEmbedding(ctx context.Context, embedding Embedding) error {
	if len(embedding.Vector) == 0 {
		return fmt.Errorf("%w: embedding vector is empty", errutils.ErrStoreInternal)
	}
	if embedding.Model == "" {
		return fmt.Errorf("%w: embedding model is empty", errutils.ErrStoreInternal)
	}

	_, err := s.client.ExecContext(ctx, upsertEmbeddingQuery,
		embedding.HardwareID,
		embedding.Model,
		len(embedding.Vector),
		embedding.SourceHash,
		encodeVector(embedding.Vector),
	)
	if err != nil {
		if errutils.IsForeignKeyViolation(err) {
			return fmt.Errorf(
				"hardware %d: %w", embedding.HardwareID, errutils.ErrStoreNotFound,
			)
		}

		return fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return nil
}

func (s *SQLiteStore) GetAllEmbeddings(ctx context.Context) ([]EmbeddedHardware, error) {
	rows := make([]embeddedHardwareRow, 0)
	if err := sqlscan.Select(ctx, s.client, &rows, getAllEmbeddingsQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	res := make([]EmbeddedHardware, 0, len(rows))
	for _, row := range rows {
		vector, err := decodeVector(row.Vector)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: hardware %d: %w", errutils.ErrStoreInternal, row.Hardware.ID, err,
			)
		}

		res = append(res, EmbeddedHardware{
			Hardware: row.Hardware,
			Model:    row.Model,
			Vector:   vector,
		})
	}

	return res, nil
}

func (s *SQLiteStore) GetEmbeddingStatus(ctx context.Context, hardwareID int) (EmbeddingStatus, error) {
	var res EmbeddingStatus
	if err := sqlscan.Get(ctx, s.client, &res, getEmbeddingStatusQuery, hardwareID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmbeddingStatus{}, fmt.Errorf(
				"hardware %d: %w", hardwareID, errutils.ErrStoreNotFound,
			)
		}

		return EmbeddingStatus{}, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

func (s *SQLiteStore) GetAllEmbeddingStatus(ctx context.Context) ([]EmbeddingStatus, error) {
	res := make([]EmbeddingStatus, 0)
	if err := sqlscan.Select(ctx, s.client, &res, getAllEmbeddingStatusQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", errutils.ErrStoreInternal, err)
	}

	return res, nil
}

type embeddedHardwareRow struct {
	Hardware
	Model  string `db:"embedding_model"`
	Vector []byte `db:"vector"`
}

func encodeVector(vector []float32) []byte {
	out := make([]byte, len(vector)*4)
	for i, value := range vector {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(value))
	}

	return out
}

func decodeVector(raw []byte) ([]float32, error) {
	if len(raw) == 0 || len(raw)%4 != 0 {
		return nil, fmt.Errorf("vector blob is %d bytes, want a positive multiple of 4", len(raw))
	}

	out := make([]float32, len(raw)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}

	return out, nil
}
