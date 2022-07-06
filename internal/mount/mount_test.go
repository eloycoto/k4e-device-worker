package mount_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	mm "github.com/project-flotta/flotta-device-worker/internal/mount"
	"github.com/project-flotta/flotta-operator/models"
	"golang.org/x/sys/unix"
)

var _ = Describe("Mount", func() {
	var (
		depMock          *util.MockIDependencies
		blockDevicePath  string
		otherBlockDevice string
		charDevicePath   string
		devFolder        string
		tmpFolder        string
		err              error
	)

	// Creating block-device
	// sudo mknod block_device b 89 1
	// creating char-device
	// sudo mknod block_device c 89 1

	Context("Mount block device", func() {
		BeforeEach(func() {

			tmpFolder, err = ioutil.TempDir("/tmp/", "mountfile")
			Expect(err).NotTo(HaveOccurred())

			depMock = &util.MockIDependencies{}

			devFolder, err = ioutil.TempDir("/tmp/", "dev")
			Expect(err).NotTo(HaveOccurred())

			// devFolder = "/dev/foodev"

			Expect(os.RemoveAll(devFolder)).To(BeNil())
			Expect(os.MkdirAll(devFolder, os.ModePerm)).NotTo(HaveOccurred())

			blockDevicePath = fmt.Sprintf("%s/blockdevice", devFolder)
			Expect(unix.Mknod(blockDevicePath, syscall.S_IFBLK|syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IWGRP, int(unix.Mkdev(7, 1)))).
				NotTo(HaveOccurred())

			otherBlockDevice = fmt.Sprintf("%s/otherblockdevice", devFolder)
			Expect(unix.Mknod(otherBlockDevice, syscall.S_IFBLK|uint32(os.FileMode(0660)), int(unix.Mkdev(7, 2)))).
				NotTo(HaveOccurred())

			charDevicePath = fmt.Sprintf("%s/chardevice", devFolder)
			Expect(unix.Mknod(charDevicePath, syscall.S_IFCHR|uint32(os.FileMode(0660)), int(unix.Mkdev(4, 1)))).
				NotTo(HaveOccurred())
		})

		AfterEach(func() {
			depMock.AssertExpectations(GinkgoT())

			// Expect(os.RemoveAll(devFolder)).To(BeNil())
			Expect(os.RemoveAll(tmpFolder)).To(BeNil())

		})

		It("mount block device without error", func() {
			// given
			dep := util.NewDependencies("/")
			mount := models.Mount{
				Device:    blockDevicePath,
				Directory: tmpFolder,
				Type:      "ext4",
			}

			configuration := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{
					Mounts: []*models.Mount{&mount},
				},
			}

			mountManager, err := mm.New()
			Expect(err).To(BeNil())

			// when
			Expect(mountManager.Update(configuration)).NotTo(HaveOccurred())

			// then
			a, mounts, err := mm.GetMounts(dep)
			Expect(err).NotTo(HaveOccurred())

			debug := func() {
				fmt.Printf("A =%+v\n", a)
				fmt.Printf("Mounts =%+v\n", mounts)

				for key := range mounts {
					fmt.Println("Key--->", key)
				}
				fmt.Println(mount.Directory)

				fmt.Printf("Err =%+v\n", err)
			}
			debug()

			_, found := mounts[mount.Directory]
			Expect(found).To(BeTrue())

			Expect(unix.Unmount(mount.Device, 0)).NotTo(HaveOccurred())
		})

		It("Don't try to mount char devices", func() {
			// given
			dep := util.NewDependencies("/")
			mount := models.Mount{
				Device:    charDevicePath,
				Directory: "/mnt",
				Type:      "ext4",
			}

			configuration := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{
					Mounts: []*models.Mount{&mount},
				},
			}

			mountManager, err := mm.New()
			Expect(err).To(BeNil())

			// when
			Expect(mountManager.Update(configuration)).To(HaveOccurred())

			// then
			_, mounts, err := mm.GetMounts(dep)
			Expect(err).To(BeNil())

			// we expect not to found the mount here
			_, found := mounts[mount.Directory]
			Expect(found).To(BeFalse())
		})

		It("Unmount a device before mounting again", func() {
			// given
			dep := util.NewDependencies("/")
			mount := models.Mount{
				Device:    blockDevicePath,
				Directory: "/mnt",
				Type:      "ext4",
			}

			configuration := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{
					Mounts: []*models.Mount{&mount},
				},
			}

			mountManager, err := mm.New()
			Expect(err).To(BeNil())

			// when
			Expect(mountManager.Update(configuration)).To(BeNil())

			// try to mount the other block device on the same folder
			mount = models.Mount{
				Device:    otherBlockDevice,
				Directory: "/mnt",
				Type:      "ext4",
			}

			// when
			Expect(mountManager.Update(configuration)).To(BeNil())

			// then
			_, mounts, err := mm.GetMounts(dep)
			Expect(err).To(BeNil())

			// we expect not to found the mount here
			m, found := mounts[mount.Directory]
			Expect(found).To(BeFalse())
			Expect(m.Device).To(Equal(otherBlockDevice))
		})

		It("Cannot mount on a folder twice", func() {
			// given
			dep := util.NewDependencies("/")
			mounts := []*models.Mount{
				{
					Device:    blockDevicePath,
					Directory: "/mnt",
					Type:      "ext4",
				},
				{
					Device:    otherBlockDevice,
					Directory: "/mnt",
					Type:      "ext4",
				},
			}

			configuration := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{
					Mounts: mounts,
				},
			}

			mountManager, err := mm.New()
			Expect(err).To(BeNil())

			// when
			Expect(mountManager.Update(configuration)).To(BeNil())

			// then
			_, mountMap, err := mm.GetMounts(dep)
			Expect(err).To(BeNil())

			// we expect not to found the mount here
			m, found := mountMap["/mnt"]
			Expect(found).To(BeFalse())
			Expect(m.Device).To(Equal(blockDevicePath))
		})

		It("Ignore non valid mount configuration", func() {
			// given
			dep := util.NewDependencies("/")
			mounts := []*models.Mount{
				{
					Device:    charDevicePath,
					Directory: "/mnt",
					Type:      "ext4",
				},
				{
					Device:    otherBlockDevice,
					Directory: "/mnt",
					Type:      "ext4",
				},
			}

			configuration := models.DeviceConfigurationMessage{
				Configuration: &models.DeviceConfiguration{
					Mounts: mounts,
				},
			}

			mountManager, err := mm.New()
			Expect(err).To(BeNil())

			// when
			Expect(mountManager.Update(configuration)).To(BeNil())

			// then
			_, mountMap, err := mm.GetMounts(dep)
			Expect(err).To(BeNil())

			// we expect not to found the mount here
			m, found := mountMap["/mnt"]
			Expect(found).To(BeFalse())
			Expect(m.Device).To(Equal(otherBlockDevice))
		})
	})
})
