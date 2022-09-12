package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type RemoveInstanceStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	operationStorage storage.Operations
}

var _ process.Step = &RemoveInstanceStep{}

func NewRemoveInstanceStep(instanceStorage storage.Instances, operationStorage storage.Operations) RemoveInstanceStep {
	return RemoveInstanceStep{
		operationManager: process.NewOperationManager(operationStorage),
		instanceStorage:  instanceStorage,
		operationStorage: operationStorage,
	}
}

func (s RemoveInstanceStep) Name() string {
	return "Remove_Instance"
}

func (s RemoveInstanceStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	if operation.Temporary {
		delay := time.Second
		instance, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			log.Errorf("cannot fetch instance with ID: %s from storage", operation.InstanceID)
			return operation, delay, err
		}

		// empty RuntimeID means there is no runtime in the Provisioner Domain
		log.Info("Removing the RuntimeID field from instance")
		instance.RuntimeID = ""
		_, err = s.instanceStorage.Update(*instance)
		if err != nil {
			log.Errorf("cannot update instance with ID: %s", instance.InstanceID)
			return operation, delay, err
		}

		log.Info("Removing the RuntimeID field from operation")
		operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.RuntimeID = ""
		}, log)

		if delay != 0 {
			return operation, delay, nil
		}

		return operation, 0, nil
	} else {
		log.Info("Removing the instance")
		delay := s.removeInstancePermanently(operation.InstanceID)
		if delay != 0 {
			return operation, delay, nil
		}

		log.Info("Removing the userID field from operation")
		operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.ProvisioningParameters.ErsContext.UserID = ""
		}, log)
		if delay != 0 {
			return operation, delay, nil
		}

		return operation, 0, nil
	}
}

func (s RemoveInstanceStep) removeInstancePermanently(instanceID string) time.Duration {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		return 10 * time.Second
	}

	return 0
}
