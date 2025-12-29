package service

import (
	"log"
	"time"
)

// SyncService handles background synchronization between local and cloud storage
type SyncService struct {
	local    Store
	cloud    Store
	interval time.Duration
	stopChan chan struct{}
	lastSync time.Time
}

// NewSyncService creates a new sync service
func NewSyncService(local, cloud Store, interval time.Duration) *SyncService {
	return &SyncService{
		local:    local,
		cloud:    cloud,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins the background sync process
func (s *SyncService) Start() {
	log.Printf("Starting sync service (interval: %v)", s.interval)
	go s.run()
}

// Stop halts the background sync process
func (s *SyncService) Stop() {
	close(s.stopChan)
	log.Println("Sync service stopped")
}

// run is the main sync loop
func (s *SyncService) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Perform initial sync
	s.performSync()

	for {
		select {
		case <-ticker.C:
			s.performSync()
		case <-s.stopChan:
			return
		}
	}
}

// performSync synchronizes entries between local and cloud storage
func (s *SyncService) performSync() {
	log.Println("Starting sync...")

	// Get all entries from both stores
	localEntries, err := s.local.GetAll()
	if err != nil {
		log.Printf("Error getting local entries: %v", err)
		return
	}

	cloudEntries, err := s.cloud.GetAll()
	if err != nil {
		log.Printf("Error getting cloud entries: %v", err)
		return
	}

	syncedCount := 0
	conflictCount := 0

	// Sync local → cloud and resolve conflicts
	for id, localEntry := range localEntries {
		cloudEntry, existsInCloud := cloudEntries[id]

		if !existsInCloud {
			// Entry only exists locally - push to cloud
			if err := s.cloud.SaveEntry(localEntry); err != nil {
				log.Printf("Error pushing entry %s to cloud: %v", id, err)
			} else {
				syncedCount++
			}
		} else {
			// Entry exists in both - resolve conflict using LastModifiedTimestamp
			if localEntry.LastModifiedTimestamp.After(cloudEntry.LastModifiedTimestamp) {
				// Local is newer - push to cloud
				if err := s.cloud.SaveEntry(localEntry); err != nil {
					log.Printf("Error updating entry %s in cloud: %v", id, err)
				} else {
					syncedCount++
				}
			} else if cloudEntry.LastModifiedTimestamp.After(localEntry.LastModifiedTimestamp) {
				// Cloud is newer - pull to local
				if err := s.local.SaveEntry(cloudEntry); err != nil {
					log.Printf("Error updating entry %s locally: %v", id, err)
				} else {
					conflictCount++
				}
			}
			// If timestamps are equal, no sync needed
		}
	}

	// Sync cloud → local for entries that only exist in cloud
	for id, cloudEntry := range cloudEntries {
		if _, existsLocally := localEntries[id]; !existsLocally {
			// Entry only exists in cloud - pull to local
			if err := s.local.SaveEntry(cloudEntry); err != nil {
				log.Printf("Error pulling entry %s from cloud: %v", id, err)
			} else {
				syncedCount++
			}
		}
	}

	s.lastSync = time.Now()
	log.Printf("Sync complete: %d entries synced, %d conflicts resolved", syncedCount, conflictCount)
}

// SyncNow triggers an immediate sync
func (s *SyncService) SyncNow() {
	s.performSync()
}

// LastSyncTime returns when the last sync occurred
func (s *SyncService) LastSyncTime() time.Time {
	return s.lastSync
}
