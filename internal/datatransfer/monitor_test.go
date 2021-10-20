package datatransfer_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/jakub-dzon/k4e-device-worker/internal/configuration"
	"github.com/jakub-dzon/k4e-device-worker/internal/datatransfer"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload/api"
	"github.com/jakub-dzon/k4e-operator/models"
)

var _ = Describe("Datatransfer", func() {

	var (
		datadir       string
		mockCtrl      *gomock.Controller
		wkManager     *workload.WorkloadManager
		configManager *configuration.Manager
		wkwMock       *workload.MockWorkloadWrapper
		err           error
		cfg           models.DeviceConfigurationMessage
	)

	BeforeEach(func() {

		datadir, err = ioutil.TempDir("", "worloadTest")
		Expect(err).ToNot(HaveOccurred())

		mockCtrl = gomock.NewController(GinkgoT())
		wkwMock = workload.NewMockWorkloadWrapper(mockCtrl)

		wkwMock.EXPECT().Init().Return(nil).AnyTimes()
		wkwMock.EXPECT().PersistConfiguration().AnyTimes()

		wkManager, err = workload.NewWorkloadManagerWithParams(datadir, wkwMock)
		Expect(err).NotTo(HaveOccurred(), "Cannot start the Workload Manager")

		configManager = configuration.NewConfigurationManager(datadir)
		fmt.Println(configManager)

		deviceConfiguration := models.DeviceConfiguration{
			Heartbeat: &models.HeartbeatConfiguration{
				PeriodSeconds: 1,
			},
			Storage: &models.StorageConfiguration{
				S3: &models.S3StorageConfiguration{
					AwsAccessKeyID:     "",
					AwsCaBundle:        "",
					AwsSecretAccessKey: "",
					BucketHost:         "",
					BucketName:         "",
					BucketPort:         0,
				},
			},
		}

		cfg = models.DeviceConfigurationMessage{
			Configuration: &deviceConfiguration,
			DeviceID:      "",
			Version:       "",
			Workloads: []*models.Workload{
				{
					Data: &models.DataConfiguration{
						Paths: []*models.DataPath{{
							Source: "/metrics",
							Target: "/metrics",
						}},
					},
					Name:          "test",
					Specification: "{}",
				},

				{
					Data: &models.DataConfiguration{
						Paths: []*models.DataPath{{
							Source: "/metrics",
							Target: "/metrics",
						}},
					},
					Name:          "foo",
					Specification: "{}",
				},
				{
					Data: &models.DataConfiguration{
						Paths: []*models.DataPath{{
							Source: "/metrics",
							Target: "/metrics",
						}},
					},
					Name:          "bar",
					Specification: "{}",
				},
			},
			WorkloadsMonitoringInterval: 0,
		}

		configManager.Update(cfg)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		_ = os.Remove(datadir)
	})

	Context("Sync", func() {

		It("work as expected", func() {

			// given
			fssync := datatransfer.NewMockFileSync(mockCtrl)
			fssync.EXPECT().Connect().Times(1)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).AnyTimes()

			monitor := datatransfer.NewMonitor(wkManager, configManager)
			monitor.SetStorage(fssync)

			wkwMock.EXPECT().List().Return([]api.WorkloadInfo{{
				Id:     "test",
				Name:   "test",
				Status: "Running",
			}}, nil).AnyTimes()
			// when
			err := monitor.ForceSync()

			// then
			Expect(err).NotTo(HaveOccurred())
		})

		It("one sync failed", func() {
			// given
			fssync := datatransfer.NewMockFileSync(mockCtrl)
			fssync.EXPECT().Connect().Times(1)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).Times(1)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed")).Times(1)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).Times(1)

			monitor := datatransfer.NewMonitor(wkManager, configManager)
			monitor.SetStorage(fssync)

			wkwMock.EXPECT().List().Return([]api.WorkloadInfo{
				{Id: "test", Name: "test", Status: "Running"},
				{Id: "foo", Name: "foo", Status: "Running"},
				{Id: "bar", Name: "bar", Status: "Running"},
			}, nil).AnyTimes()

			// when
			err := monitor.ForceSync()

			// then
			Expect(err).To(HaveOccurred())
			Expect(monitor.GetLastSuccessfulSyncTime("test")).NotTo(BeNil())
			Expect(monitor.GetLastSuccessfulSyncTime("foo")).To(BeNil())
			Expect(monitor.GetLastSuccessfulSyncTime("bar")).NotTo(BeNil())
		})

		It("does not report workloads without path", func() {
			// given
			fssync := datatransfer.NewMockFileSync(mockCtrl)
			fssync.EXPECT().Connect().Times(1)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).Times(1)

			monitor := datatransfer.NewMonitor(wkManager, configManager)
			monitor.SetStorage(fssync)

			wkwMock.EXPECT().List().Return([]api.WorkloadInfo{
				{Id: "test", Name: "test", Status: "Running"},
				{Id: "invalid", Name: "invalid", Status: "Running"},
			}, nil).AnyTimes()

			// when
			err := monitor.ForceSync()

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(monitor.GetLastSuccessfulSyncTime("test")).NotTo(BeNil())
		})

		It("does nothing if there are no workloads", func() {

			// given
			fssync := datatransfer.NewMockFileSync(mockCtrl)
			fssync.EXPECT().Connect().Times(0)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).Times(0)

			monitor := datatransfer.NewMonitor(wkManager, configManager)
			monitor.SetStorage(fssync)

			wkwMock.EXPECT().List().Return([]api.WorkloadInfo{}, nil).AnyTimes()

			// when
			err := monitor.ForceSync()

			// then
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Context("WorkloadRemoved", func() {
		It("work as expected", func() {
			// given
			fssync := datatransfer.NewMockFileSync(mockCtrl)
			fssync.EXPECT().Connect().Times(1)
			fssync.EXPECT().SyncPath(gomock.Any(), gomock.Any()).Times(1)

			monitor := datatransfer.NewMonitor(wkManager, configManager)
			monitor.SetStorage(fssync)

			wkwMock.EXPECT().List().Return([]api.WorkloadInfo{
				{Id: "test", Name: "test", Status: "Running"},
			}, nil).AnyTimes()

			// when
			monitor.WorkloadRemoved("test")

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(monitor.GetLastSuccessfulSyncTime("test")).To(BeNil())
		})
	})

	Context("HasStorage", func() {
		It("true if it's defined", func() {
			// given
			monitor := datatransfer.NewMonitor(wkManager, configManager)

			// when
			res := monitor.HasStorageDefined()

			// then
			Expect(res).To(BeTrue())
		})

		It("false no storage defined", func() {
			// given
			monitor := datatransfer.NewMonitor(wkManager, configManager)
			cfg.Configuration.Storage = nil
			configManager.Update(cfg)

			// when
			res := monitor.HasStorageDefined()

			// then
			Expect(res).To(BeFalse())
		})

		It("false no s3-storage defined", func() {
			// given
			monitor := datatransfer.NewMonitor(wkManager, configManager)
			cfg.Configuration.Storage.S3 = nil
			configManager.Update(cfg)

			// when
			res := monitor.HasStorageDefined()

			// then
			Expect(res).To(BeFalse())
		})

	})

})
