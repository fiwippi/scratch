package halo

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nfnt/resize"
	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

type store struct {
	pool *sqlx.DB
}

func newStore(path string) (*store, error) {
	pool, err := sqlx.Connect("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(1)
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS photos (
			id        TEXT PRIMARY KEY,
			data      BLOB NOT NULL,
			timestamp DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tags (
			name TEXT PRIMARY KEY
		);`,
		`CREATE TABLE IF NOT EXISTS img_tags (
			img_id   TEXT NOT NULL,
			tag_name TEXT NOT NULL,

			-- Relationships
			PRIMARY KEY (img_id, tag_name),
			FOREIGN KEY (img_id)
				REFERENCES photos (id)
                	ON UPDATE CASCADE 
                	ON DELETE CASCADE
			FOREIGN KEY (tag_name)
				REFERENCES tags (name)
                	ON UPDATE CASCADE 
                	ON DELETE CASCADE
		);`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(stmt); err != nil {
			return nil, err
		}
	}

	return &store{
		pool: pool,
	}, nil
}

func (s *store) Close() error {
	return s.pool.Close()
}

// Tags

func (s *store) GetTags() ([]string, error) {
	var names []string
	return names, s.pool.Select(&names, `SELECT name FROM tags ORDER BY name ASC`)
}

func (s *store) AddTag(name string) error {
	return s.runTx(func(tx *sqlx.Tx) error {
		return txAddTag(tx, name)
	})
}

func (s *store) DeleteTag(name string) error {
	return s.runTx(func(tx *sqlx.Tx) error {
		return txDeleteTag(tx, name)
	})
}

func (s *store) HasTag(name string) bool {
	var exists bool
	s.runTx(func(tx *sqlx.Tx) error {
		exists = txHasTag(tx, name)
		return nil
	})
	return exists
}

// Images

func (s *store) GetImageIDsWithTag(tag string) ([]ulid.ULID, error) {
	stmt := `
		SELECT p.id FROM photos p
		JOIN img_tags it ON p.id = it.img_id
		WHERE it.tag_name = ?
		ORDER BY p.timestamp ASC`
	var ids []ulid.ULID
	return ids, s.pool.Select(&ids, stmt, tag)
}

type newImageData struct {
	img       image.Image
	timestamp time.Time
}

func (s *store) AddImages(images []newImageData, tags ...string) ([]ulid.ULID, error) {
	ids := make([]ulid.ULID, len(images))
	data := make([][]byte, len(images))

	for i, imgData := range images {
		ids[i] = ulid.Make()

		bounds := imgData.img.Bounds()
		w, h := uint(bounds.Dx()), uint(bounds.Dy())

		var buf bytes.Buffer
		if w <= 900 && h <= 900 {
			if err := jpeg.Encode(&buf, imgData.img, nil); err != nil {
				return nil, fmt.Errorf("encode img %d: %w", i+1, err)
			}
		} else {
			if w > h {
				w, h = min(1000, w), 0
			} else {
				w, h = 0, min(1000, h)
			}
			if err := jpeg.Encode(&buf, resize.Resize(w, h, imgData.img, resize.Lanczos3), nil); err != nil {
				return nil, fmt.Errorf("encode img %d: %w", i+1, err)
			}
		}
		data[i] = buf.Bytes()
	}

	err := s.runTx(func(tx *sqlx.Tx) error {
		for i := range images {
			stmt := `INSERT INTO photos (id, data, timestamp) VALUES (?, ?, ?)`
			if _, err := tx.Exec(stmt, ids[i], data[i], images[i].timestamp); err != nil {
				return err
			}
			if len(tags) > 0 {
				for _, t := range tags {
					if err := txAddTagToImage(tx, t, ids[i]); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("exec tx: %w", err)
	}
	return ids, nil
}

func (s *store) DeleteImage(id ulid.ULID) error {
	_, err := s.pool.Exec(`DELETE FROM photos WHERE id = ?`, id)
	return err
}

func (s *store) GetImageData(id ulid.ULID) ([]byte, error) {
	var data []byte
	return data, s.pool.Get(&data, `SELECT data FROM photos WHERE id = ?`, id)
}

// Tx Helpers

type txFunc func(tx *sqlx.Tx) error

func (s *store) runTx(fn txFunc) error {
	tx, err := s.pool.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func txAddTag(tx *sqlx.Tx, name string) error {
	_, err := tx.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, name)
	return err
}

func txDeleteTag(tx *sqlx.Tx, name string) error {
	_, err := tx.Exec(`DELETE FROM tags WHERE name = ?`, name)
	return err
}

func txHasTag(tx *sqlx.Tx, name string) bool {
	var exists bool
	tx.Get(&exists, `SELECT COUNT(*) > 0 FROM tags WHERE name = ?`, name)
	return exists
}

func txAddTagToImage(tx *sqlx.Tx, name string, id ulid.ULID) error {
	// Always add the tag since we assume
	// it might not previously exist
	if err := txAddTag(tx, name); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO img_tags (img_id, tag_name) VALUES (?, ?)`, id, name)
	return err
}
