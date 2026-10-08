package goose

import (
	"database/sql"
)

func (c *Client) Up(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		current, err := c.GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				return nil
			}
			return err
		}

		if err = c.runMigration(db, next, migrateUp); err != nil {
			return err
		}
	}
}

// UpMissing applies every known migration that is not currently applied,
// in version order. Unlike Up, it also fills gaps below the latest applied
// version. This is useful for deployments where migrations were applied out
// of order or where a previous migration run stopped after applying only part
// of the known migrations.
func (c *Client) UpMissing(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	// Ensure the migration status table exists before reading its records.
	if _, err := c.GetDBVersion(db); err != nil {
		return err
	}

	rows, err := db.Query("SELECT version_id, is_applied FROM " + c.TableName + " ORDER BY id ASC")
	if err != nil {
		return err
	}
	latestStatus := make(map[int64]bool)
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			rows.Close()
			return err
		}
		latestStatus[version] = isApplied
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, migration := range migrations {
		if latestStatus[migration.Version] {
			continue
		}
		if err := c.runMigration(db, migration, migrateUp); err != nil {
			return err
		}
		latestStatus[migration.Version] = true
	}
	return nil
}

func (c *Client) UpByOne(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := c.GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
	if err != nil {
		return err
	}

	if err = c.runMigration(db, next, migrateUp); err != nil {
		return err
	}

	return nil
}
