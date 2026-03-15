# ec2-go-operator

A Kubernetes operator for managing AWS EC2 instances declaratively.

## What it does

Manages EC2 instance lifecycle (create, delete, drift detection) through Kubernetes custom resources.

## Prerequisites

- Go 1.24+
- Kubernetes cluster (or kind/minikube)
- AWS account with EC2 permissions
- kubectl

## Quick Start

```bash
# Install CRDs
make install

# Set AWS credentials
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"

# Run the operator
make run
```

## Usage

Create an EC2 instance:

```yaml
apiVersion: compute.mycloud.com/v1
kind: EC2Instance
metadata:
  name: myinstance
spec:
  amiID: "ami-02dfbd4ff395f2a1b"
  instanceType: "t3.medium"
  region: "us-east-1"
  keyPair: "my-key"
  subnet: "subnet-xxxxx"
  securityGroups:
    - "sg-xxxxx"
```

```bash
kubectl apply -f Sample/ec2instance.yaml
kubectl get ec2instance myinstance
```

Delete the instance:

```bash
kubectl delete ec2instance myinstance
```

## Features

- ✅ Instance creation with AWS SDK v2
- ✅ Automatic cleanup with finalizers
- ✅ Drift detection (checks if instance still exists)
- ✅ Status updates with public IP, instance ID, state
- ✅ Pretty kubectl output columns

## Development

```bash
make build          # Build binary
make test           # Run tests
make manifests      # Generate CRDs
```

## License

Apache 2.0
