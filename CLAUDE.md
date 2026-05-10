# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

A Kubernetes operator built with [Kubebuilder](https://book.kubebuilder.io/) that manages AWS EC2 instances as Kubernetes custom resources. The CRD is `EC2Instance` (API group `compute.mycloud.com/v1`). The operator reconciles desired state (EC2InstanceSpec) against actual AWS state, including drift detection and finalizer-based deletion.

## Commands

```bash
# Build the manager binary
make build

# Run the controller locally against the current kubeconfig cluster
make run

# Run unit tests (uses envtest, not real AWS)
make test

# Run a single test package
go test ./internal/controller/... -v -run TestName

# Generate CRD manifests and RBAC from markers after editing types
make manifests

# Regenerate DeepCopy methods after editing api/v1/ec2instance_types.go
make generate

# Lint
make lint
make lint-fix

# Install CRDs into cluster
make install

# Deploy controller to cluster
make deploy IMG=<your-image>

# Run e2e tests (requires Kind)
make test-e2e
```

## Architecture

### Reconciliation flow

[internal/controller/ec2instance_controller.go](internal/controller/ec2instance_controller.go) contains the main loop. Execution path on each reconcile:

1. Fetch `EC2Instance` — if not found (deleted), exit cleanly.
2. If `DeletionTimestamp` is set → call `deleteInstance` → remove finalizer `ec2instance.compute.mycloud.com` → exit.
3. If `Status.InstanceID` is non-empty → call `checkEC2InstanceExists` to detect drift:
   - On AWS error: clear all status fields, requeue (triggers a new create on next loop).
   - If instance not running: set `Status.State = "Unknown"`, clear PublicIP.
   - If running and state was `Unknown`: recover state fields.
4. If `Status.InstanceID` is empty → add finalizer (if missing), then call `createEC2Instance` → populate status.

The controller requeues with a 1-second delay after status updates because status writes themselves trigger re-reconciliation.

### Key files

| File | Purpose |
|---|---|
| [api/v1/ec2instance_types.go](api/v1/ec2instance_types.go) | CRD spec/status types; add fields here then run `make generate manifests` |
| [internal/controller/ec2instance_controller.go](internal/controller/ec2instance_controller.go) | Reconciliation loop |
| [internal/controller/createinstance.go](internal/controller/createinstance.go) | Calls `RunInstances`, waits for `running` state (3 min timeout), returns `CreatedInstanceInfo` |
| [internal/controller/deleteInstance.go](internal/controller/deleteInstance.go) | Calls `TerminateInstances`, waits for `terminated` state (5 min timeout) |
| [internal/controller/checkInstance.go](internal/controller/checkInstance.go) | Drift detection — checks if instance exists and is in `running` state |
| [internal/controller/aws_client.go](internal/controller/aws_client.go) | Builds EC2 client from `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` env vars (static credentials, not IAM roles) |
| [cmd/main.go](cmd/main.go) | Manager setup — metrics, leader election, health probes |

### CRD type structure

`EC2InstanceSpec` required fields: `amiID`, `instanceType`, `region`. Optional: `availabilityZone`, `keyPair`, `securityGroups`, `subnet`, `userData`, `tags`, `storage` (root + additional volumes), `associatePublicIP`.

`EC2InstanceStatus` is populated by the controller: `instanceID`, `state`, `publicIP`, `publicDNS`, `privateIP`, `privateDNS`, `launchTime`.

`kubectl get ec2instances` shows custom columns: InstanceType, State, PublicIP, InstanceID (defined via `+kubebuilder:printcolumn` markers).

### Testing

Unit tests ([internal/controller/ec2instance_controller_test.go](internal/controller/ec2instance_controller_test.go)) use Ginkgo v2 + envtest (fake Kubernetes API). The AWS client is **not mocked**, so tests that reach AWS calls will fail with `InvalidAMIID.Malformed` — this is known/expected behavior in the current codebase. E2e tests ([test/e2e/](test/e2e/)) deploy to a Kind cluster and test operator pod readiness and metrics endpoint.

### Immutable infrastructure model

EC2 instances managed by this operator are treated as immutable. Spec changes do not trigger in-place updates — the operator only creates on first reconcile and deletes when the CR is deleted. Drift recovery is limited to detecting if a running instance disappears and marking state accordingly.
