package workload_test

import (
	"fmt"
	"io/ioutil"

	gomock "github.com/golang/mock/gomock"
	"github.com/jakub-dzon/k4e-device-worker/internal/workload"
	api "github.com/jakub-dzon/k4e-device-worker/internal/workload/api"
	"github.com/jakub-dzon/k4e-operator/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {

	var (
		datadir   = "/tmp"
		mockCtrl  *gomock.Controller
		wkManager *workload.WorkloadManager
		wkwMock   *workload.MockWorkloadWrapper
		err       error
	)

	BeforeEach(func() {
		datadir, err = ioutil.TempDir("", "worloadTest")
		Expect(err).ToNot(HaveOccurred())

		mockCtrl = gomock.NewController(GinkgoT())
		wkwMock = workload.NewMockWorkloadWrapper(mockCtrl)

		wkwMock.EXPECT().Init().Return(nil).AnyTimes()
		wkManager, err = workload.NewWorkloadManagerWithParams(datadir, wkwMock)
		Expect(err).NotTo(HaveOccurred(), "Cannot start the Workload Manager")

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Update", func() {
		It("works as expected", func() {

			workloads := []*models.Workload{}

			for i := 0; i < 10; i++ {
				wkName := fmt.Sprintf("test%d", i)
				workloads = append(workloads, &models.Workload{
					Data:          &models.DataConfiguration{},
					Name:          wkName,
					Specification: "{}",
				})
				wkwMock.EXPECT().Remove(wkName).Times(1)
			}

			// given
			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
				DeviceID:      "",
				Version:       "",
				Workloads:     workloads,
			}

			wkwMock.EXPECT().List().AnyTimes()
			wkwMock.EXPECT().Run(gomock.Any(), gomock.Any()).AnyTimes()

			// when
			errors := wkManager.Update(cfg)

			// then
			Expect(errors).To(HaveLen(0))
		})

		It("Workload Run failed", func() {
			// given
			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
				DeviceID:      "",
				Version:       "",
				Workloads: []*models.Workload{{
					Data:          &models.DataConfiguration{},
					Name:          "test",
					Specification: "{}",
				}},
			}

			wkwMock.EXPECT().List().AnyTimes()
			wkwMock.EXPECT().Remove("test").AnyTimes()
			wkwMock.EXPECT().Run(gomock.Any(), gomock.Any()).Return(fmt.Errorf("Cannot run workload")).Times(1)

			// when
			errors := wkManager.Update(cfg)

			// then
			Expect(errors).To(HaveLen(1))
		})

		It("Some worklodas failed", func() {
			// So make sure that all worksloads tried to be executed, even if one
			// failed.

			// given
			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
				DeviceID:      "",
				Version:       "",
				Workloads: []*models.Workload{
					{
						Data:          &models.DataConfiguration{},
						Name:          "test",
						Specification: "{}",
					},
					{
						Data:          &models.DataConfiguration{},
						Name:          "testB",
						Specification: "{}",
					},
				},
			}

			wkwMock.EXPECT().List().AnyTimes()
			wkwMock.EXPECT().Remove("test").AnyTimes()
			wkwMock.EXPECT().Run(gomock.Any(), getManifest(datadir, "test")).Return(fmt.Errorf("Cannot run workload")).Times(1)

			wkwMock.EXPECT().Remove("testB").AnyTimes()
			wkwMock.EXPECT().Run(gomock.Any(), getManifest(datadir, "testB")).Return(nil).Times(1)

			// when
			errors := wkManager.Update(cfg)

			// then
			Expect(errors).To(HaveLen(1))
		})

		It("staled workload got deleted if it's not in the config", func() {
			// given
			cfg := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{Heartbeat: &models.HeartbeatConfiguration{PeriodSeconds: 1}},
				DeviceID:      "",
				Version:       "",
				Workloads: []*models.Workload{
					{
						Data:          &models.DataConfiguration{},
						Name:          "test",
						Specification: "{}",
					},
					{
						Data:          &models.DataConfiguration{},
						Name:          "testB",
						Specification: "{}",
					},
				},
			}

			currentWorkloads := []api.WorkloadInfo{
				{Id: "stale", Name: "stale", Status: "running"},
			}
			wkwMock.EXPECT().List().Return(currentWorkloads, nil).AnyTimes()

			wkwMock.EXPECT().Remove("test").AnyTimes()
			wkwMock.EXPECT().Remove("testB").AnyTimes()
			wkwMock.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			wkwMock.EXPECT().Remove("stale").Times(1)

			// when
			errors := wkManager.Update(cfg)

			// then
			Expect(errors).To(HaveLen(0))
		})

	})
})

func getManifest(datadir string, workloadName string) string {
	return fmt.Sprintf("%s/manifests/%s.yaml", datadir, workloadName)
}
