package halo

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/jmoiron/sqlx"
	"github.com/nfnt/resize"
	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

type store struct {
	pool *sqlx.DB
}

const inMemory string = "file::memory:"

func newStore(path string) (*store, error) {
	pool, err := sqlx.Connect("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(1)
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS originals (
			id   TEXT PRIMARY KEY,
			data BLOB NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS thumbnails (
			id   TEXT PRIMARY KEY,
			data BLOB NOT NULL,

			-- Relationships
			FOREIGN KEY (id) 
		    	REFERENCES originals (id)
                	ON UPDATE CASCADE 
                	ON DELETE CASCADE 
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
				REFERENCES originals (id)
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

func (s *store) RenameTag(oldName, newName string) error {
	return s.runTx(func(tx *sqlx.Tx) error {
		if !txHasTag(tx, newName) {
			_, err := tx.Exec(`UPDATE tags SET name = ? WHERE name = ?`, newName, oldName)
			return err
		}
		if _, err := tx.Exec(`UPDATE img_tags SET tag_name = ? WHERE tag_name = ?`, newName, oldName); err != nil {
			return err
		}
		_, err := tx.Exec(`DELETE FROM tags WHERE name = ?`, oldName)
		return err
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

func (s *store) AddTagToImage(name string, id ulid.ULID) error {
	return s.runTx(func(tx *sqlx.Tx) error {
		return txAddTagToImage(tx, name, id)
	})
}

func (s *store) DeleteTagFromImage(name string, id ulid.ULID) error {
	_, err := s.pool.Exec(`DELETE FROM img_tags WHERE img_id = ? AND tag_name = ?`, id, name)
	return err
}

// Images

func (s *store) GetImageIDs(tags ...string) ([]ulid.ULID, error) {
	var ids []ulid.ULID
	if len(tags) == 0 {
		return ids, s.pool.Select(&ids, `SELECT id FROM originals ORDER BY id ASC`)
	}
	stmt := `
		SELECT img_id FROM img_tags WHERE tag_name IN (?)
		GROUP BY img_id HAVING COUNT(DISTINCT tag_name) = ?
		ORDER BY img_id ASC`
	stmt, args, err := sqlx.In(stmt, tags, len(tags))
	if err != nil {
		return nil, err
	}
	return ids, s.pool.Select(&ids, stmt, args...)
}

func (s *store) GetAssociatedImageTags(id ulid.ULID) ([]string, error) {
	var tags []string
	return tags, s.pool.Select(&tags,
		`SELECT tag_name FROM img_tags WHERE img_id = ? ORDER BY tag_name ASC`, id)
}

func (s *store) GetUnassociatedImageTags(id ulid.ULID) ([]string, error) {
	var tags []string
	return tags, s.pool.Select(&tags,
		`SELECT name FROM tags t WHERE NOT EXISTS (
			SELECT 1 FROM img_tags it
			WHERE it.img_id = ? AND it.tag_name = t.name
		)
		ORDER BY name ASC`, id)
}

func (s *store) AddImages(images []image.Image, tags ...string) ([]ulid.ULID, error) {
	ids := make([]ulid.ULID, len(images))
	originals := make([][]byte, len(images))
	thumbnails := make([][]byte, len(images))

	for i, img := range images {
		ids[i] = ulid.Make()

		// So, it's more space efficient if we convert both
		// the original and the thumbnail image to JPEG
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, nil); err != nil {
			return nil, fmt.Errorf("encode original %d: %w", i+1, err)
		}
		originals[i] = buf.Bytes()

		buf = bytes.Buffer{} // Need a different backing array for the thumbnail
		bounds := img.Bounds()
		// We're kinda assuming that images will never
		// be smaller than 250 in width or height,
		// otherwise it doesn't make sense to resize
		w, h := uint(bounds.Dx()), uint(bounds.Dy())
		if w > h {
			w, h = min(350, w), 0
		} else {
			w, h = 0, min(350, h)
		}
		if err := jpeg.Encode(&buf, resize.Resize(w, h, img, resize.Lanczos3), nil); err != nil {
			return nil, fmt.Errorf("encode thumbnail %d: %w", i+1, err)
		}
		thumbnails[i] = buf.Bytes()
	}

	err := s.runTx(func(tx *sqlx.Tx) error {
		for i := range images {
			if _, err := tx.Exec(`INSERT INTO originals (id, data) VALUES (?, ?)`, ids[i], originals[i]); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO thumbnails (id, data) VALUES (?, ?)`, ids[i], thumbnails[i]); err != nil {
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
	_, err := s.pool.Exec(`DELETE FROM originals WHERE id = ?`, id)
	return err
}

func (s *store) GetOriginalImageBytes(id ulid.ULID) ([]byte, error) {
	var data []byte
	return data, s.pool.Get(&data, `SELECT data FROM originals WHERE id = ?`, id)
}

func (s *store) GetThumbnailImageBytes(id ulid.ULID) ([]byte, error) {
	var data []byte
	return data, s.pool.Get(&data, `SELECT data FROM thumbnails WHERE id = ?`, id)
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
