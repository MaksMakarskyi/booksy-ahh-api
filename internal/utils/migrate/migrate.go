package migrate

import (
	"database/sql"
	"fmt"

	"github.com/MaksMakarskyi/booksy-go-api/migrations"
	"github.com/pressly/goose/v3"
)

func Up(db *sql.DB, tableName string) error {
	goose.SetBaseFS(migrations.FS)
	goose.SetTableName(tableName)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
