package heartbeat_test

import (
	"github.com/golang/mock/gomock"
	"github.com/jakub-dzon/k4e-device-worker/internal/configuration"
	"github.com/jakub-dzon/k4e-device-worker/internal/datatransfer"
	"github.com/jakub-dzon/k4e-device-worker/internal/hardware"
	"github.com/jakub-dzon/k4e-device-worker/internal/heartbeat"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload"
	"github.com/jakub-dzon/k4e-operator/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heartbeat", func() {

	var (
		datadir       = "/tmp"
		mockCtrl      *gomock.Controller
		wkManager     *workload.WorkloadManager
		configManager *configuration.Manager
		hw            = &hardware.Hardware{}
		monitor       = &datatransfer.Monitor{}
		hb            = &heartbeat.Heartbeat{}
		err           error
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		wkwMock := workload.NewMockWorkloadWrapper(mockCtrl)

		wkwMock.EXPECT().Init().Return(nil).AnyTimes()
		wkwMock.EXPECT().List().AnyTimes()
		wkwMock.EXPECT().PersistConfiguration().AnyTimes()

		wkManager, err = workload.NewWorkloadManagerWithParams(datadir, wkwMock)
		Expect(err).To(BeNil(), "Cannot start the Workload Manager")

		configManager = configuration.NewConfigurationManager(datadir)
		hb = heartbeat.NewHeartbeatService(nil,
			configManager,
			wkManager,
			hw,
			monitor)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Start", func() {
		It("Ticker is stopped if it's not started", func() {
			Expect(hb.HasStarted()).To(BeFalse(), "Ticker is initialized when it shouldn't")
			hb.Start()
			Expect(hb.HasStarted()).To(BeTrue())
		})
	})

	Context("Update", func() {

		It("Ticker is created", func() {
			Expect(hb.HasStarted()).To(BeFalse(), "Ticker is initialized when it shouldn't")

			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
				DeviceID:      "",
				Version:       "",
				Workloads:     []*models.Workload{},
			}

			Expect(hb.Update(cfg)).To(BeNil(), "Cannot update ticker")
			Expect(hb.HasStarted()).To(BeTrue())
		})

		It("Ticker not created on invalid config", func() {
			Expect(hb.HasStarted()).To(BeFalse(), "Ticker is initialized when it shouldn't")

			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{},
				DeviceID:      "",
				Version:       "",
				Workloads:     []*models.Workload{},
			}

			Expect(hb.Update(cfg)).To(BeNil(), "Cannot update ticker")
			Expect(hb.HasStarted()).To(BeTrue())
		})

	})

})
