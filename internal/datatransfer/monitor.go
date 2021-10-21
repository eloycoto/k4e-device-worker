package datatransfer

import (
	"fmt"
	"path"
	"sync"
	"time"

	"git.sr.ht/~spc/go-log"
	"github.com/hashicorp/go-multierror"
	"github.com/jakub-dzon/k4e-device-worker/internal/configuration"
	"github.com/jakub-dzon/k4e-device-worker/internal/datatransfer/s3"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload"
	"github.com/jakub-dzon/k4e-operator/models"
)

type Monitor struct {
	workloads                   *workload.WorkloadManager
	config                      *configuration.Manager
	ticker                      *time.Ticker
	lastSuccessfulSyncTimes     map[string]time.Time
	lastSuccessfulSyncTimesLock sync.RWMutex
	fsSync                      FileSync
}

func NewMonitor(workloadsManager *workload.WorkloadManager, configManager *configuration.Manager) *Monitor {
	ticker := time.NewTicker(configManager.GetDataTransferInterval())

	monitor := Monitor{
		workloads:                   workloadsManager,
		config:                      configManager,
		ticker:                      ticker,
		lastSuccessfulSyncTimes:     make(map[string]time.Time),
		lastSuccessfulSyncTimesLock: sync.RWMutex{},
	}
	return &monitor
}

func (m *Monitor) Start() {
	go func() {
		for range m.ticker.C {
			m.syncPaths()
		}
		log.Info("The monitor was stopped")
	}()
}

func (m *Monitor) Deregister() error {
	log.Info("Stopping monitor ticker")
	if m.ticker != nil {
		m.ticker.Stop()
	}
	return nil
}

func (m *Monitor) GetLastSuccessfulSyncTime(workloadName string) *time.Time {
	m.lastSuccessfulSyncTimesLock.RLock()
	defer m.lastSuccessfulSyncTimesLock.RUnlock()
	if t, ok := m.lastSuccessfulSyncTimes[workloadName]; ok {
		return &t
	}
	return nil
}

func (m *Monitor) WorkloadRemoved(workloadName string) {
	m.syncPathsWorkload(workloadName)

	m.lastSuccessfulSyncTimesLock.Lock()
	defer m.lastSuccessfulSyncTimesLock.Unlock()
	delete(m.lastSuccessfulSyncTimes, workloadName)
}

func (m *Monitor) syncPathsWorkload(workloadName string) {
	if !m.HasStorageDefined() {
		return
	}

	dataPaths := m.getPathsOfWorkload(workloadName)
	if len(dataPaths) == 0 {
		return
	}

	err := m.connectFS()
	if err != nil {
		log.Errorf("Error while creating s3 synchronizer: %v", err)
		return
	}

	hostPath := m.workloads.GetExportedHostPath(workloadName)
	success := true
	for _, dp := range dataPaths {
		source := path.Join(hostPath, dp.Source)
		target := dp.Target
		log.Debugf("Synchronizing [device]%s => [remote]%s", source, target)

		if err := m.fsSync.SyncPath(source, target); err != nil {
			log.Errorf("Error while synchronizing [device]%s => [remote]%s: %v", source, target, err)
			success = false
		}
	}
	if success {
		m.storeLastUpdateTime(workloadName)
	}
}

func (m *Monitor) getPathsOfWorkload(workloadName string) []*models.DataPath {
	dataPaths := []*models.DataPath{}
	for _, wd := range m.config.GetWorkloads() {
		if wd.Name == workloadName {
			if wd.Data != nil && len(wd.Data.Paths) > 0 {
				dataPaths = wd.Data.Paths
			}
			break
		}
	}
	return dataPaths
}

func (m *Monitor) ForceSync() error {
	return m.syncPaths()
}

func (m *Monitor) HasStorageDefined() bool {
	storage := m.config.GetDeviceConfiguration().Storage
	if storage == nil {
		return false
	}

	return storage.S3 != nil
}

func (m *Monitor) InitStorage() error {
	storage := m.config.GetDeviceConfiguration().Storage
	s3sync, err := s3.NewSync(*storage.S3)
	if err != nil {
		return err
	}
	m.SetStorage(s3sync)
	return nil
}

func (m *Monitor) SetStorage(fs FileSync) {
	m.fsSync = fs
}

func (m *Monitor) connectFS() error {
	return m.fsSync.Connect()
}

func (m *Monitor) syncPaths() error {

	workloads, err := m.workloads.ListWorkloads()
	if err != nil {
		log.Errorf("Can't get the list of workloads: %v", err)
		return err
	}

	if len(workloads) == 0 {
		log.Trace("No worloads to return")
		return nil
	}

	ok := m.HasStorageDefined()
	if !ok {
		log.Trace("Monitor does not have storage defined")
		return nil
	}

	err = m.connectFS()
	if err != nil {
		return err
	}

	workloadToDataPaths := make(map[string][]*models.DataPath)
	for _, wd := range m.config.GetWorkloads() {
		if wd.Data != nil && len(wd.Data.Paths) > 0 {
			workloadToDataPaths[wd.Name] = wd.Data.Paths
		}
	}

	var errors error
	// Monitor actual workloads and not ones expected by the configuration

	for _, wd := range workloads {
		dataPaths := workloadToDataPaths[wd.Name]
		if len(dataPaths) == 0 {
			continue
		}

		hostPath := m.workloads.GetExportedHostPath(wd.Name)
		success := true
		for _, dp := range dataPaths {
			source := path.Join(hostPath, dp.Source)
			target := dp.Target

			logMessage := fmt.Sprintf("synchronizing [device]%s => [remote]%s", source, target)
			log.Debug(logMessage)

			if err := m.fsSync.SyncPath(source, target); err != nil {
				errors = multierror.Append(errors, fmt.Errorf("Error while %s", logMessage))
				log.Errorf("Error while %s", logMessage)
				success = false
			}
		}

		if success {
			m.storeLastUpdateTime(wd.Name)
		}
	}
	return errors
}

func (m *Monitor) storeLastUpdateTime(workloadName string) {
	m.lastSuccessfulSyncTimesLock.Lock()
	defer m.lastSuccessfulSyncTimesLock.Unlock()
	m.lastSuccessfulSyncTimes[workloadName] = time.Now()
}

func (m *Monitor) Update(configuration models.DeviceConfigurationMessage) error {
	storage := configuration.Configuration.Storage
	s3sync, err := s3.NewSync(*storage.S3)
	if err != nil {
		return fmt.Errorf("Observer update failed: %s", err)
	}
	m.SetStorage(s3sync)
	return nil
}
