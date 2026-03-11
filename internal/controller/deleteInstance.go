package controller

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	computev1 "github.com/gargmanik6080/ec2-go-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func deleteInstance(ctx context.Context, ec2Instance *computev1.EC2Instance) (bool, error) {
	l := log.FromContext(ctx)

	l.Info("Deleting EC2 Instance", "instanceID", ec2Instance.Status.InstanceID)

	ec2Client := awsClient(ec2Instance.Spec.Region)

	// Terminating the instance
	terminateResult, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	})

	if err != nil {
		l.Error(err, "Failed to terminate the instance")
		return false, err
	}

	l.Info("Instance termination initiated",
		"instanceID", ec2Instance.Status.InstanceID,
		"currentStatus", terminateResult.TerminatingInstances[0].CurrentState.Name)

	// Wait for the instance to be terminated
	waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
	maxWaitTime := 5 * time.Minute
	waitParams := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	}

	l.Info("Waiting for the instance to be terminated",
		"instanceID", ec2Instance.Status.InstanceID,
		"maxWaitTime", maxWaitTime)

	// Waiting
	err = waiter.Wait(ctx, waitParams, maxWaitTime)

	if err != nil {
		l.Error(err, "Failed while waiting for instance termination",
			"instanceID", ec2Instance.Status.InstanceID)
		return false, err	
	}

	l.Info("EC2 instance successfully terminated", "instanceID", ec2Instance.Status.InstanceID)
	return true, nil
}