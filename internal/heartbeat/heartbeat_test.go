package heartbeat_test

import (
	"context"

	"github.com/golang/mock/gomock"
	"github.com/jakub-dzon/k4e-device-worker/internal/configuration"
	"github.com/jakub-dzon/k4e-device-worker/internal/datatransfer"
	"github.com/jakub-dzon/k4e-device-worker/internal/hardware"
	"github.com/jakub-dzon/k4e-device-worker/internal/heartbeat"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload"
	"github.com/jakub-dzon/k4e-operator/models"
	"google.golang.org/grpc"

	pb "github.com/redhatinsights/yggdrasil/protocol"

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
		client        DispatcherInstance
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
		client = DispatcherInstance{}
		hb = heartbeat.NewHeartbeatService(&client,
			configManager,
			wkManager,
			hw,
			monitor)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("HeartBeatData test", func() {
		It("report empty workloads an up status", func() {
			hbData := heartbeat.NewHeartbeatData(configManager, wkManager, hw, monitor)
			heartbeatInfo := hbData.RetrieveInfo()
			Expect(heartbeatInfo.Status).To(Equal("up"))
			Expect(heartbeatInfo.Workloads).To(BeEmpty())
		})

		// @TODO Workloads need more work,because WorkloadManager.Update needs a
		// lot more than a simple Name/spec
		// It("report workload correctly", func() {
		// 	cfg := models.DeviceConfigurationMessage{
		// 		Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
		// 		DeviceID:      "",
		// 		Version:       "",
		// 		Workloads: []*models.Workload{{
		// 			Data:          &models.DataConfiguration{},
		// 			Name:          "test",
		// 			Specification: "test",
		// 		}},
		// 	}

		// 	wkManager.Update(cfg)
		// 	hbData := heartbeat.NewHeartbeatData(configManager, wkManager, hw, monitor)
		// 	heartbeatInfo := hbData.RetrieveInfo()
		// 	Expect(heartbeatInfo.Status).To(Equal("up"))
		// 	Expect(heartbeatInfo.Workloads).To(BeEmpty())
		// })

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

// We keep the latest send data to make sure that we validate the data sent to
// the operator without sent at all
type DispatcherInstance struct {
	latestData *pb.Data
}

func (d *DispatcherInstance) Send(ctx context.Context, in *pb.Data, opts ...grpc.CallOption) (*pb.Receipt, error) {
	d.latestData = in
	return nil, nil
}

func (d *DispatcherInstance) Register(ctx context.Context, in *pb.RegistrationRequest, opts ...grpc.CallOption) (*pb.RegistrationResponse, error) {
	return nil, nil
}
