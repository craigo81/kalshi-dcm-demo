// Package persistence provides file-based data persistence for the DCM demo.
// Core Principle 18: Supports 5-year retention through file-based storage.
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// PERSISTENCE MANAGER
// CP 18: Recordkeeping with configurable retention
// =============================================================================

// Manager handles file-based persistence
type Manager struct {
	dataDir     string
	enabled     bool
	saveInterval time.Duration
	mu          sync.Mutex
}

// DataSnapshot represents the full store state for persistence
type DataSnapshot struct {
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`

	// User data
	Users       map[string]*models.User       `json:"users"`
	UsersByEmail map[string]string            `json:"users_by_email"`

	// KYC records
	KYCRecords  map[string]*models.KYCRecord `json:"kyc_records"`

	// Wallets and transactions
	Wallets      map[string]*models.Wallet       `json:"wallets"`
	Transactions map[string]*models.Transaction  `json:"transactions"`
	TxByWallet   map[string][]string             `json:"tx_by_wallet"`

	// Orders and positions
	Orders       map[string]*models.Order    `json:"orders"`
	OrdersByUser map[string][]string         `json:"orders_by_user"`
	Positions    map[string]*models.Position `json:"positions"`
	PositionsByUser map[string][]string      `json:"positions_by_user"`

	// Compliance
	Alerts       []models.ComplianceAlert          `json:"alerts"`
	Halts        map[string]*models.EmergencyHalt  `json:"halts"`

	// Counters
	IDCounter   int64 `json:"id_counter"`
}

// AuditArchive holds audit entries for a specific time period
// CP 18: Separate audit files for efficient retention management
type AuditArchive struct {
	StartDate time.Time           `json:"start_date"`
	EndDate   time.Time           `json:"end_date"`
	Entries   []models.AuditEntry `json:"entries"`
}

// NewManager creates a new persistence manager
func NewManager(dataDir string, enabled bool) (*Manager, error) {
	if enabled {
		// Create data directory if it doesn't exist
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}

		// Create subdirectories
		subdirs := []string{"snapshots", "audit", "archive"}
		for _, subdir := range subdirs {
			path := filepath.Join(dataDir, subdir)
			if err := os.MkdirAll(path, 0755); err != nil {
				return nil, fmt.Errorf("failed to create %s directory: %w", subdir, err)
			}
		}
	}

	return &Manager{
		dataDir:      dataDir,
		enabled:      enabled,
		saveInterval: 5 * time.Minute, // Auto-save every 5 minutes
	}, nil
}

// =============================================================================
// SNAPSHOT OPERATIONS
// =============================================================================

// SaveSnapshot persists the current store state to disk
func (m *Manager) SaveSnapshot(snapshot *DataSnapshot) error {
	if !m.enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot.Version = "1.0"
	snapshot.Timestamp = time.Now().UTC()

	// Create timestamped filename
	filename := fmt.Sprintf("snapshot_%s.json", snapshot.Timestamp.Format("20060102_150405"))
	path := filepath.Join(m.dataDir, "snapshots", filename)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	// Also update "latest" symlink/file
	latestPath := filepath.Join(m.dataDir, "snapshots", "latest.json")
	if err := os.WriteFile(latestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write latest snapshot: %w", err)
	}

	return nil
}

// LoadLatestSnapshot loads the most recent snapshot from disk
func (m *Manager) LoadLatestSnapshot() (*DataSnapshot, error) {
	if !m.enabled {
		return nil, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	latestPath := filepath.Join(m.dataDir, "snapshots", "latest.json")

	data, err := os.ReadFile(latestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No snapshot exists yet
		}
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snapshot DataSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

// =============================================================================
// AUDIT LOG OPERATIONS
// CP 18: 5-year retention with monthly archives
// =============================================================================

// SaveAuditEntries appends audit entries to the current month's log
func (m *Manager) SaveAuditEntries(entries []models.AuditEntry) error {
	if !m.enabled || len(entries) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Group entries by month
	entriesByMonth := make(map[string][]models.AuditEntry)
	for _, entry := range entries {
		monthKey := entry.Timestamp.Format("2006-01")
		entriesByMonth[monthKey] = append(entriesByMonth[monthKey], entry)
	}

	// Append to each month's file
	for monthKey, monthEntries := range entriesByMonth {
		filename := fmt.Sprintf("audit_%s.json", monthKey)
		path := filepath.Join(m.dataDir, "audit", filename)

		// Load existing entries
		var existing []models.AuditEntry
		if data, err := os.ReadFile(path); err == nil {
			var archive AuditArchive
			if err := json.Unmarshal(data, &archive); err == nil {
				existing = archive.Entries
			}
		}

		// Append new entries
		existing = append(existing, monthEntries...)

		// Save updated archive
		archive := AuditArchive{
			StartDate: time.Date(
				monthEntries[0].Timestamp.Year(),
				monthEntries[0].Timestamp.Month(),
				1, 0, 0, 0, 0, time.UTC,
			),
			EndDate: time.Now().UTC(),
			Entries: existing,
		}

		data, err := json.MarshalIndent(archive, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal audit archive: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write audit archive: %w", err)
		}
	}

	return nil
}

// LoadAuditEntries loads audit entries within a date range
// CP 18: Supports queries across retention period
func (m *Manager) LoadAuditEntries(since, until time.Time) ([]models.AuditEntry, error) {
	if !m.enabled {
		return nil, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var allEntries []models.AuditEntry
	auditDir := filepath.Join(m.dataDir, "audit")

	// Iterate through months in range
	current := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)

	for current.Before(end) {
		monthKey := current.Format("2006-01")
		filename := fmt.Sprintf("audit_%s.json", monthKey)
		path := filepath.Join(auditDir, filename)

		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read audit file %s: %w", filename, err)
			}
			current = current.AddDate(0, 1, 0)
			continue
		}

		var archive AuditArchive
		if err := json.Unmarshal(data, &archive); err != nil {
			return nil, fmt.Errorf("failed to unmarshal audit file %s: %w", filename, err)
		}

		// Filter entries within date range
		for _, entry := range archive.Entries {
			if !entry.Timestamp.Before(since) && entry.Timestamp.Before(until) {
				allEntries = append(allEntries, entry)
			}
		}

		current = current.AddDate(0, 1, 0)
	}

	return allEntries, nil
}

// ArchiveOldAuditLogs moves audit logs older than retention period to archive
// CP 18: Maintains 5-year retention with archive capability
func (m *Manager) ArchiveOldAuditLogs(retentionYears int) error {
	if !m.enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(-retentionYears, 0, 0)
	auditDir := filepath.Join(m.dataDir, "audit")
	archiveDir := filepath.Join(m.dataDir, "archive")

	entries, err := os.ReadDir(auditDir)
	if err != nil {
		return fmt.Errorf("failed to read audit directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !isAuditFile(entry.Name()) {
			continue
		}

		// Parse month from filename (audit_2024-01.json)
		monthStr := entry.Name()[6:13] // Extract "2024-01"
		fileMonth, err := time.Parse("2006-01", monthStr)
		if err != nil {
			continue
		}

		// Archive if older than cutoff
		if fileMonth.Before(cutoff) {
			oldPath := filepath.Join(auditDir, entry.Name())
			newPath := filepath.Join(archiveDir, entry.Name())

			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to archive %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// isAuditFile checks if filename matches audit file pattern
func isAuditFile(name string) bool {
	return len(name) > 6 && name[:6] == "audit_" && filepath.Ext(name) == ".json"
}

// =============================================================================
// CLEANUP OPERATIONS
// =============================================================================

// CleanOldSnapshots removes snapshots older than specified days, keeping latest
func (m *Manager) CleanOldSnapshots(keepDays int) error {
	if !m.enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -keepDays)
	snapshotDir := filepath.Join(m.dataDir, "snapshots")

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "latest.json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(snapshotDir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove old snapshot %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// =============================================================================
// STATISTICS
// =============================================================================

// Stats returns storage statistics
type Stats struct {
	SnapshotCount   int       `json:"snapshot_count"`
	AuditFileCount  int       `json:"audit_file_count"`
	ArchiveCount    int       `json:"archive_count"`
	TotalSizeBytes  int64     `json:"total_size_bytes"`
	OldestAudit     time.Time `json:"oldest_audit"`
	LatestSnapshot  time.Time `json:"latest_snapshot"`
}

// GetStats returns storage statistics
func (m *Manager) GetStats() (*Stats, error) {
	if !m.enabled {
		return nil, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stats := &Stats{}

	// Count snapshots
	snapshotDir := filepath.Join(m.dataDir, "snapshots")
	if entries, err := os.ReadDir(snapshotDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && e.Name() != "latest.json" {
				stats.SnapshotCount++
				if info, err := e.Info(); err == nil {
					stats.TotalSizeBytes += info.Size()
					if info.ModTime().After(stats.LatestSnapshot) {
						stats.LatestSnapshot = info.ModTime()
					}
				}
			}
		}
	}

	// Count audit files
	auditDir := filepath.Join(m.dataDir, "audit")
	if entries, err := os.ReadDir(auditDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && isAuditFile(e.Name()) {
				stats.AuditFileCount++
				if info, err := e.Info(); err == nil {
					stats.TotalSizeBytes += info.Size()
				}
				// Parse date for oldest
				monthStr := e.Name()[6:13]
				if t, err := time.Parse("2006-01", monthStr); err == nil {
					if stats.OldestAudit.IsZero() || t.Before(stats.OldestAudit) {
						stats.OldestAudit = t
					}
				}
			}
		}
	}

	// Count archives
	archiveDir := filepath.Join(m.dataDir, "archive")
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				stats.ArchiveCount++
				if info, err := e.Info(); err == nil {
					stats.TotalSizeBytes += info.Size()
				}
			}
		}
	}

	return stats, nil
}
