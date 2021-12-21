package server

import (
	"context"
	"encoding/json"

	"git.sr.ht/~spc/go-log"
	configuration2 "github.com/jakub-dzon/k4e-device-worker/internal/configuration"
	registration2 "github.com/jakub-dzon/k4e-device-worker/internal/registration"
	"github.com/jakub-dzon/k4e-operator/models"
	pb "github.com/redhatinsights/yggdrasil/protocol"
)

type deviceServer struct {
	pb.UnimplementedWorkerServer
	configManager       *configuration2.Manager
	registrationManager *registration2.Registration
	deviceID            string
}

func NewDeviceServer(configManager *configuration2.Manager, registrationManager *registration2.Registration, deviceID string) *deviceServer {
	return &deviceServer{
		configManager:       configManager,
		registrationManager: registrationManager,
		deviceID:            deviceID,
	}
}

// Send implements the "Send" method of the Worker gRPC service.
func (s *deviceServer) Send(ctx context.Context, d *pb.Data) (*pb.Receipt, error) {

	go func() {
		deviceConfigurationMessage := models.DeviceConfigurationMessage{}
		err := json.Unmarshal(d.Content, &deviceConfigurationMessage)
		if err != nil {
			log.Warnf("Cannot unmarshal message: %v", err)
		}
		err = s.configManager.Update(deviceConfigurationMessage)
		if err != nil {
			log.Warnf("Failed to process message: %v", err)
		}
	}()

	// Respond to the start request that the work was accepted.
	return &pb.Receipt{}, nil
}

// Send implements the "Send" method of the Worker gRPC service.
func (s *deviceServer) Events(ctx context.Context, data *pb.APIResponse) (*pb.Empty, error) {
	// Got 401, registation is not send, so create a new Registration request to the operator.
	if data.StatusCode == 401 {
		s.registrationManager.RegisterDevice()
	}

	if data.StatusCode == 200 {
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("STATUS CODE %+v", data.StatusCode)
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
		log.Errorf("--------------------------------")
	}
	log.Errorf("Got event here %+v", data)
	return nil, nil
}

// Disconnect implements the "Disconnect" method of the Worker gRPC service.
func (s *deviceServer) Disconnect(ctx context.Context, in *pb.Empty) (*pb.DisconnectResponse, error) {
	log.Info("received worker disconnect request")
	err := s.registrationManager.Deregister()
	if err != nil {
		log.Warnf("Cannot disconnect: %v", err)
	}
	// Respond to the disconnect request that the work was accepted.
	return &pb.DisconnectResponse{}, nil

}
