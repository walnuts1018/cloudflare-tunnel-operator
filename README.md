# cloudflare-tunnel-operator

## Development

### Prerequisites

- go version v1.23.3+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- aqua version 2.25.1+
  - `brew install aquaproj/aqua/aqua`

### Install Dependencies

```shell
aqua i
```

### Start Cluster

```shell
make start
tilt up --host 0.0.0.0
```

### Stop Cluster

```shell
make stop
```
