# EC2 Go Operator

A Kubernetes operator for managing AWS EC2 instances declaratively using the operator pattern. Built with Kubebuilder and AWS SDK v2.

> **Note**: This is a learning/demonstration project showcasing operator development with Kubebuilder. While fully functional for basic EC2 management, it's not production-hardened. See [Production Readiness](#production-readiness) for details.

## Overview

This operator demonstrates how to extend Kubernetes with custom resources to manage AWS EC2 instances as native Kubernetes objects. It showcases core operator patterns including declarative management, automatic reconciliation, drift detection, and proper cleanup handling through finalizers.

**Purpose**: This project serves as a practical example of building Kubernetes operators with Kubebuilder, integrating with cloud provider APIs, and implementing the controller pattern. It's suitable for learning, demonstrations, and as a starting point for more robust implementations.

### Key Capabilities

- **Declarative Instance Management**: Define EC2 instances using Kubernetes manifests
- **Automatic Reconciliation**: Continuously ensures actual state matches desired state
- **Drift Detection**: Monitors instance state and detects manual changes outside Kubernetes
- **Graceful Cleanup**: Uses finalizers to ensure proper resource deletion
- **Status Reporting**: Real-time status updates with instance details (IP, state, DNS)
- **kubectl Integration**: Custom columns for easy resource inspection

## Architecture

The operator follows the standard Kubernetes controller pattern:

1. **Custom Resource Definition (CRD)**: Defines the `EC2Instance` resource schema
2. **Controller**: Watches for EC2Instance resources and reconciles state
3. **Reconciliation Loop**:
   - Detects when EC2Instance resources are created/updated/deleted
   - Creates EC2 instances via AWS API when needed
   - Updates status with instance information
   - Handles deletion with proper cleanup using finalizers
   - Performs drift detection to ensure instances exist

### Components

```
api/v1/
  ├── ec2instance_types.go    # CRD definitions and types
internal/controller/
  ├── ec2instance_controller.go  # Main reconciliation logic
  ├── createinstance.go          # EC2 instance creation
  ├── deleteinstance.go          # EC2 instance termination
  ├── checkinstance.go           # Drift detection
  └── aws_client.go              # AWS SDK client initialization
```

## Prerequisites

- **Go**: 1.24 or higher
- **Kubernetes Cluster**: v1.28+ (or local cluster like kind/minikube)
- **AWS Account**: With EC2 permissions (RunInstances, TerminateInstances, DescribeInstances)
- **kubectl**: Configured to access your cluster
- **AWS Credentials**: Access key and secret key with EC2 permissions

### Required AWS IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:TerminateInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus"
      ],
      "Resource": "*"
    }
  ]
}
```

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/gargmanik6080/ec2-go-operator.git
cd ec2-go-operator
```

### 2. Install CRDs

```bash
make install
```

This installs the EC2Instance custom resource definition in your cluster.

### 3. Configure AWS Credentials

```bash
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
```

### 4. Run the Operator

**Option A: Run locally (development)**
```bash
make run
```

**Option B: Deploy to cluster**
```bash
make docker-build docker-push IMG=<your-registry>/ec2-go-operator:latest
make deploy IMG=<your-registry>/ec2-go-operator:latest
```

## Usage

### Create an EC2 Instance

Create a YAML manifest defining your EC2 instance:

```yaml
apiVersion: compute.mycloud.com/v1
kind: EC2Instance
metadata:
  name: my-web-server
  namespace: default
spec:
  amiID: "ami-02dfbd4ff395f2a1b"           # Amazon Linux 2023
  instanceType: "t3.medium"
  region: "us-east-1"
  keyPair: "my-key-pair"
  subnet: "subnet-0abc1234def567890"
  securityGroups:
    - "sg-0abc1234def567890"
  tags:
    environment: "production"
    team: "platform"
  storage:
    rootVolume:
      size: 20
      type: "gp3"
```

Apply the manifest:

```bash
kubectl apply -f ec2instance.yaml
```

### Check Instance Status

```bash
# List all EC2 instances
kubectl get ec2instance

# Output:
# NAME            INSTANCETYPE   STATE     PUBLICIP         INSTANCEID
# my-web-server   t3.medium      running   54.123.45.67     i-0abc1234def567890

# Get detailed information
kubectl describe ec2instance my-web-server

# Watch for status changes
kubectl get ec2instance my-web-server -w
```

### Update an Instance

The operator currently uses an immutable infrastructure approach - instances cannot be updated in place. To make changes:

1. Delete the existing instance
2. Modify the spec
3. Create a new instance

> **Note**: In-place updates would require additional logic to detect spec changes, determine which fields can be updated vs require replacement, and implement the update operations. This is intentionally not implemented in this demonstration project.

### Delete an Instance

```bash
kubectl delete ec2instance my-web-server
```

The operator will:
1. Terminate the EC2 instance in AWS
2. Wait for termination to complete
3. Remove the finalizer
4. Delete the Kubernetes resource

## Configuration

### EC2Instance Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amiID` | string | Yes | Amazon Machine Image ID |
| `instanceType` | string | Yes | EC2 instance type (e.g., t3.medium) |
| `region` | string | Yes | AWS region (e.g., us-east-1) |
| `availabilityZone` | string | No | Specific AZ within region |
| `keyPair` | string | No | SSH key pair name |
| `securityGroups` | []string | No | List of security group IDs |
| `subnet` | string | No | Subnet ID for the instance |
| `userData` | string | No | User data script (base64) |
| `tags` | map[string]string | No | Key-value tags for the instance |
| `storage.rootVolume` | VolumeConfig | No | Root volume configuration |
| `associatePublicIP` | bool | No | Assign public IP address |

### Status Fields

The operator populates these status fields:

| Field | Description |
|-------|-------------|
| `instanceID` | AWS instance ID (e.g., i-0abc123...) |
| `state` | Current instance state (pending, running, terminated) |
| `publicIP` | Public IPv4 address |
| `publicDNS` | Public DNS name |
| `privateIP` | Private IPv4 address |
| `privateDNS` | Private DNS name |
| `launchTime` | Instance launch timestamp |

## Features

### ✅ Instance Lifecycle Management

- **Creation**: Launches EC2 instances via AWS SDK v2
- **Monitoring**: Waits for instances to reach running state
- **Status Updates**: Continuously updates resource status
- **Deletion**: Gracefully terminates instances and cleans up

### ✅ Drift Detection

The operator checks if instances still exist in AWS during reconciliation:
- Detects manually terminated instances
- Updates status to reflect actual state
- Clears instance ID from status if instance no longer exists
- Note: Does not automatically recreate deleted instances (would require additional logic)

### ✅ Finalizers

Prevents resource deletion until AWS cleanup completes:
- Adds finalizer `ec2instance.compute.mycloud.com` on creation
- Terminates EC2 instance when Kubernetes resource is deleted
- Removes finalizer only after successful termination

### ✅ Custom kubectl Columns

Enhanced `kubectl get` output with relevant information:

```bash
kubectl get ec2instance
# NAME        INSTANCETYPE   STATE     PUBLICIP       INSTANCEID
# instance1   t3.medium      running   54.123.45.67   i-0abc123...
# instance2   t3.small       pending   <pending>      i-0def456...
```

## Development

### Project Structure

```
ec2-go-operator/
├── api/v1/                    # API definitions
│   └── ec2instance_types.go   # EC2Instance CRD types
├── internal/controller/       # Controller implementation
│   ├── ec2instance_controller.go
│   ├── createinstance.go
│   ├── deleteinstance.go
│   ├── checkinstance.go
│   └── aws_client.go
├── config/                    # Kubernetes manifests
│   ├── crd/                   # CRD definitions
│   ├── rbac/                  # RBAC policies
│   └── manager/               # Operator deployment
├── Sample/                    # Example manifests
└── Makefile                   # Build and development tasks
```

### Build Commands

```bash
# Build the operator binary
make build

# Run tests
make test

# Generate CRD manifests
make manifests

# Generate Go code (DeepCopy, etc.)
make generate

# Run locally against configured cluster
make run

# Install CRDs into cluster
make install

# Uninstall CRDs from cluster
make uninstall

# Build and push Docker image
make docker-build docker-push IMG=<registry>/ec2-go-operator:tag
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test ./... -coverprofile cover.out

# Run specific package tests
go test ./internal/controller -v
```

### Adding New Features

1. Update CRD types in `api/v1/ec2instance_types.go`
2. Run `make generate manifests` to regenerate code
3. Update controller logic in `internal/controller/`
4. Add tests
5. Run `make test` to verify
6. Run `make install` to update CRDs in cluster

## Troubleshooting

### Instance Not Creating

**Check operator logs:**
```bash
kubectl logs -f deployment/ec2-go-operator-controller-manager -n ec2-go-operator-system
```

**Common issues:**
- Invalid AWS credentials
- Missing IAM permissions
- Invalid AMI ID for the region
- Invalid subnet/security group IDs
- Insufficient EC2 capacity in the region

### Instance Stuck in Pending

**Check AWS console** for instance state and status checks

**Verify configuration:**
```bash
kubectl describe ec2instance <name>
```

### Finalizer Preventing Deletion

If an instance won't delete, you can manually remove the finalizer (use with caution):

```bash
kubectl patch ec2instance <name> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

This removes the Kubernetes resource but may leave AWS resources running.

### Drift Detection Not Working

Ensure the operator is running and has AWS credentials:
```bash
kubectl get pods -n ec2-go-operator-system
kubectl logs <operator-pod> -n ec2-go-operator-system
```

## Production Readiness

### What's Implemented ✅

- **Core Operator Functionality**: Complete reconciliation loop with create/delete/drift detection
- **Finalizers**: Proper cleanup with Kubernetes finalizers
- **Status Subresources**: Real-time status updates
- **AWS SDK v2 Integration**: Modern AWS API integration
- **Custom Print Columns**: Enhanced kubectl output
- **Basic Error Handling**: Requeue on failures

### What's Missing for Production ⚠️

This operator demonstrates core concepts but lacks several production requirements:

- ❌ **Testing**: No unit tests, integration tests, or E2E tests
- ❌ **Observability**: No Prometheus metrics, structured logging, or tracing
- ❌ **Security**: Hardcoded AWS credentials (should use IAM roles/Secrets), no validation webhooks
- ❌ **Reliability**: No leader election, exponential backoff, or circuit breakers
- ❌ **Operations**: No health checks, resource limits, or rate limiting for AWS APIs
- ❌ **Features**: Immutable instances only (no in-place updates), limited EC2 feature support

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgments

This project was built following the tutorial [Building Kubernetes Operators with Kubebuilder](https://www.youtube.com/watch?v=X5kkrIPr5Hk) by [KubeSimplify](https://www.youtube.com/@kubesimplify), with additional enhancements and modifications.

Built with:
- [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) - Kubernetes operator framework
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) - AWS API client
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller libraries
