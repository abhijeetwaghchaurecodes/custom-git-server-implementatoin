package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ImplementationRecord stores one MF service implementation in SQLite.
type ImplementationRecord struct {
	ID          int64
	Module      string // e.g. "mfcreate"
	PackageName string // e.g. "mfcreate"
	ImportPath  string // e.g. "project2-mf-implementations/mfcreate"
	StructName  string // e.g. "MFCreateImpl"
	Interface   string // e.g. "MFCreate"  (matches registry.RegisterMF*)
	SourceCode  string
	Checksum    string
	Status      string
	GoVersion   string
	InjectedAt  *time.Time
	CreatedAt   time.Time
}

// Store wraps a SQLite connection.
type Store struct{ db *sql.DB }

// NewStore opens (or creates) the SQLite DB and runs schema migrations.
func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	s := &Store{db: db}
	return s, s.migrate()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS implementations (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			module       TEXT NOT NULL,
			package_name TEXT NOT NULL,
			import_path  TEXT NOT NULL,
			struct_name  TEXT NOT NULL,
			interface    TEXT NOT NULL,
			source_code  TEXT NOT NULL,
			checksum     TEXT NOT NULL,
			status       TEXT NOT NULL DEFAULT 'pending',
			go_version   TEXT NOT NULL DEFAULT '1.21',
			injected_at  DATETIME,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`)
	return err
}

// Save inserts a new implementation record.
func (s *Store) Save(r ImplementationRecord) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO implementations
			(module, package_name, import_path, struct_name, interface,
			 source_code, checksum, status, go_version)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		r.Module, r.PackageName, r.ImportPath, r.StructName,
		r.Interface, r.SourceCode, r.Checksum, r.Status, r.GoVersion,
	)
	if err != nil {
		return 0, fmt.Errorf("insert %s: %w", r.Module, err)
	}
	return res.LastInsertId()
}

// LoadValidated returns all records with status="validated".
func (s *Store) LoadValidated() ([]ImplementationRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, module, package_name, import_path, struct_name, interface,
		       source_code, checksum, status, go_version, injected_at, created_at
		FROM implementations WHERE status = 'validated' ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ImplementationRecord
	for rows.Next() {
		var r ImplementationRecord
		var injAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Module, &r.PackageName, &r.ImportPath,
			&r.StructName, &r.Interface, &r.SourceCode, &r.Checksum,
			&r.Status, &r.GoVersion, &injAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if injAt.Valid {
			r.InjectedAt = &injAt.Time
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkInjected stamps the injected_at field for a record.
func (s *Store) MarkInjected(id int64) error {
	_, err := s.db.Exec(`UPDATE implementations SET injected_at=? WHERE id=?`, time.Now(), id)
	return err
}

func (s *Store) Close() error { return s.db.Close() }
