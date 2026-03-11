package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	computev1 "github.com/gargmanik6080/ec2-go-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createEC2Instance(ec2Instance *computev1.EC2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {
	l := log.Log.WithName("createEC2Instance")

	l.Info("=== STARTING EC2 INSTANCE CREATION ===",
		"ami", ec2Instance.Spec.AmiID,
		"instanceType", ec2Instance.Spec.InstanceType,
		"region", ec2Instance.Spec.Region)

	// creating the client for EC2 instance
	ec2Client := awsClient(ec2Instance.Spec.Region)

	// creating the input for the runInstances call
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AmiID),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		KeyName:      aws.String(ec2Instance.Spec.KeyPair),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		SecurityGroupIds: []string{ec2Instance.Spec.SecurityGroups[0]},
	}

	// Run the instance
	l.Info("=== Calling AWS RunInstances API ===")
	result, err := ec2Client.RunInstances(context.TODO(), runInput)
	if err != nil {
		l.Error(err, "Failed to create the EC2 Instance(s)")
		return nil, fmt.Errorf("Failed to create the EC2 Instance(s): %w", err)
	}

	if len(result.Instances) == 0 {
		l.Error(nil, "No instances returned in RunInstancesOutput")
		fmt.Println("No instances returned in RunInstancesOutput")
		return nil, nil
	}

	// Instance is created
	inst := result.Instances[0]
	l.Info("=== EC2 INSTANCE CREATED SUCCESSFULLY ===", "instanceID", *inst.InstanceId)

	// Waiting for the EC2 to reach running state
	l.Info("=== WAITING FOR THE INSTANCE TO BE RUNNING ===")

	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute

	err = runWaiter.Wait(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}, maxWaitTime)

	if err != nil {
		l.Error(err, "Failed to wait for the instance to be running")
		return nil, fmt.Errorf("Failed to wait for the instance to be running: %w", err)
	}

	// Now that the EC2 is running, we can fetch the details of this instance
	l.Info("=== CALLING AWS DescribeInstance API to get Instance details ===")
	describeInput := &ec2.DescribeInstancesInput{
		
	}

}
