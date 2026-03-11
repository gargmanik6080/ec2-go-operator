package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	computev1 "github.com/gargmanik6080/ec2-go-operator/api/v1"
)

func checkEC2InstanceExists(ctx context.Context, instanceID string, ec2Instance *computev1.EC2Instance) (bool, *ec2types.Instance, error) {

	fmt.Println("Checking instance: ", instanceID)
	ec2Client := awsClient(ec2Instance.Spec.Region)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
		Filters: []ec2types.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	}
	
	result, err := ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			return false, nil, nil
		}
		return false, nil, err
	}

	fmt.Println("Length of Reservations is ", len(result.Reservations))

	// Checking if there are any instances
	if len(result.Reservations) == 0 {
		return false, nil, nil
	}
	return true, &result.Reservations[0].Instances[0], nil
}