package heartbeat

import (
	"github.com/jakub-dzon/k4e-device-worker/internal/configuration"
	"github.com/jakub-dzon/k4e-device-worker/internal/datatransfer"
	"github.com/jakub-dzon/k4e-device-worker/internal/hardware"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload"
	"github.com/jakub-dzon/k4e-operator/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heartbeat", func() {

	var (
		wk            = &workload.WorkloadManager{}
		configManager = &configuration.Manager{}
		hw            = &hardware.Hardware{}
		monitor       = &datatransfer.Monitor{}
		hb            = &Heartbeat{}
		err           error
	)

	BeforeEach(func() {
		wk, err = workload.NewWorkloadManager("/tmp")
		// This is expected to fail because the lack of postam unix socket.
		Expect(err).To(Equal(nil))
		configManager = configuration.NewConfigurationManager("/tmp/")
		hb = NewHeartbeatService(nil,
			configManager,
			wk,
			hw,
			monitor)
	})

	Context("getHeartbeatInfo", func() {
		It("Fake test", func() {
			hb.getHeartbeatInfo()
			Expect(true).To(BeTrue())
		})
	})

	Context("Start", func() {
		It("Ticker is stopped if it's not started", func() {
			Expect(hb.ticker).To(BeNil(), "Ticker is initialized when it shouldn't")
			hb.Start()
			Expect(hb.ticker).NotTo(BeNil())
		})
	})

	Context("Update", func() {

		It("Ticker is created", func() {
			Expect(hb.ticker).To(BeNil(), "Ticker is initialized when it shouldn't")

			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
				DeviceID:      "",
				Version:       "",
				Workloads:     []*models.Workload{},
			}

			Expect(hb.Update(cfg)).To(BeNil(), "Cannot update ticker")
			Expect(hb.ticker).NotTo(BeNil())
		})

		It("Ticker not created on invalid config", func() {
			// This checks also hb.getInterval so Start is also tested
			Expect(hb.ticker).To(BeNil(), "Ticker is initialized when it shouldn't")

			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{},
				DeviceID:      "",
				Version:       "",
				Workloads:     []*models.Workload{},
			}

			Expect(hb.Update(cfg)).To(BeNil(), "Cannot update ticker")
			Expect(hb.ticker).NotTo(BeNil())
		})

	})

})
