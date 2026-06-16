// Package store is the local SQLite cache and offline full-text search (FTS5)
// over fetched ServiceNow metadata. It uses the pure-Go zombiezen SQLite
// driver (no cgo), which compiles FTS5 in by default — so the binary
// cross-compiles cleanly with CGO disabled.
package store

import (
	"fmt"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Store is a single-connection handle to the local cache database.
type Store struct {
	conn *sqlite.Conn
}

// Open opens (creating if needed) the database at path and ensures the schema
// exists. Use ":memory:" for an ephemeral store.
func Open(path string) (*Store, error) {
	conn, err := sqlite.OpenConn(path, sqlite.OpenReadWrite|sqlite.OpenCreate|sqlite.OpenWAL)
	if err != nil {
		return nil, fmt.Errorf("open store %q: %w", path, err)
	}
	s := &Store{conn: conn}
	if err := s.init(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}

// init creates the schema. The FTS5 virtual table doubles as a fail-fast check
// that the driver was built with FTS5 support.
func (s *Store) init() error {
	const schema = `
CREATE VIRTUAL TABLE IF NOT EXISTS metadata_fts USING fts5(
    table_name,  -- the ServiceNow table this record describes/belongs to
    kind,        -- artifact kind: field | form | business_rule | acl | flow | ...
    name,        -- record name/identifier
    body         -- searchable text (label, script, condition, ...)
);`
	if err := sqlitex.ExecuteScript(s.conn, schema, nil); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

// Index inserts a single searchable metadata record.
func (s *Store) Index(tableName, kind, name, body string) error {
	const q = `INSERT INTO metadata_fts (table_name, kind, name, body) VALUES (?, ?, ?, ?);`
	return sqlitex.Execute(s.conn, q, &sqlitex.ExecOptions{
		Args: []any{tableName, kind, name, body},
	})
}

// DeleteTable removes all cached rows for a table (called before re-indexing
// it, so repeated explorations don't accumulate duplicates).
func (s *Store) DeleteTable(tableName string) error {
	return sqlitex.Execute(s.conn, `DELETE FROM metadata_fts WHERE table_name = ?;`, &sqlitex.ExecOptions{
		Args: []any{tableName},
	})
}

// SearchHit is one offline-search result.
type SearchHit struct {
	TableName string `json:"table_name"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

// Search runs an FTS5 MATCH query and returns the hits.
func (s *Store) Search(query string) ([]SearchHit, error) {
	const q = `SELECT table_name, kind, name FROM metadata_fts WHERE metadata_fts MATCH ? ORDER BY rank;`
	var hits []SearchHit
	err := sqlitex.Execute(s.conn, q, &sqlitex.ExecOptions{
		Args: []any{query},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			hits = append(hits, SearchHit{
				TableName: stmt.ColumnText(0),
				Kind:      stmt.ColumnText(1),
				Name:      stmt.ColumnText(2),
			})
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("search %q: %w", query, err)
	}
	return hits, nil
}

// Close closes the underlying connection.
func (s *Store) Close() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}
