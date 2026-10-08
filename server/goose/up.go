package goose

import (
	"database/sql"
	"log"
)

// Up applies every known migration that is not yet applied, in ascending
// version order.
//
// A missing migration numbered below the newest applied version (typically
// one shipped in a patch release after later-numbered migrations were already
// applied) is applied out of order, but only if its version is at or above
// MinOutOfOrderVersion. This keeps long-standing gaps on old databases from
// suddenly being applied against a much newer schema. When
// MinOutOfOrderVersion is 0, out-of-order application is disabled and only
// migrations above the newest applied version run (legacy behavior).
func (c *Client) Up(db *sql.DB, dir string) error {
	pending, err := c.pendingMigrations(db, dir)
	if err != nil {
		return err
	}

	for _, m := range pending {
		if err := c.runMigration(db, m, migrateUp); err != nil {
			return err
		}
	}

	return nil
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


// UpByOne applies the lowest-versioned migration that Up would apply.
//
// It intentionally doesn't use GetDBVersion: after an out-of-order
// application, the newest version record is not the highest applied version,
// so walking forward from it would re-run already-applied migrations.
func (c *Client) UpByOne(db *sql.DB, dir string) error {
	pending, err := c.pendingMigrations(db, dir)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return ErrNoNextVersion
	}

	return c.runMigration(db, pending[0], migrateUp)
}

// pendingMigrations returns the migrations Up would apply, in ascending
// version order: every known migration that is not yet applied and is
// eligible under the out-of-order rules described in Up.
func (c *Client) pendingMigrations(db *sql.DB, dir string) (Migrations, error) {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return nil, err
	}

	applied, err := c.AppliedVersions(db)
	if err != nil {
		return nil, err
	}

	var maxApplied int64
	for v := range applied {
		if v > maxApplied {
			maxApplied = v
		}
	}

	var pending Migrations
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		if m.Version < maxApplied && (c.MinOutOfOrderVersion == 0 || m.Version < c.MinOutOfOrderVersion) {
			log.Printf("goose: skipping missing migration %d: older than the newest applied migration and below the out-of-order minimum", m.Version)
			continue
		}
		pending = append(pending, m)
	}

	return pending, nil
}
