package store

import "database/sql"

// GetSetting returns the value for a settings key, or ErrNotFound if absent.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return value, err
}

// SetSetting upserts a settings key/value pair.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		   ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}
