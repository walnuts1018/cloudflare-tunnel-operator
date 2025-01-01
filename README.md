# cloudflare-tunnel-operator

## Install

1. Create `values.yaml`
```yaml
cloudflareToken:
  cloudflareAccountID: ""
  cloudflareZoneID: ""
  cloudflareAPIToken: ""

  # If you want to get the cloudflareAPIToken from an existingSecret, include theSecret name in existingSecret.
  # You should use `cloudflareAPIToken` for the Key.
  # existingSecret: ""
```

2. Install
```shell
helm repo add cloudflare-tunnel-operator https://walnuts1018.github.io/cloudflare-tunnel-operator/
helm install cloudflare-tunnel-operator -n cloudflare-tunnel-operator --create-namespace -f values.yaml  cloudflare-tunnel-operator/cloudflare-tunnel-operator
```

3. Create `CloudflareTunnel` Resource

```yaml
apiVersion: cf-tunnel-operator.walnuts.dev/v1beta1
kind: CloudflareTunnel
metadata:
  name: cloudflaretunnel-sample
spec:
  replicas: 2
  default: true # If set to true, Cloudflare Tunnel will be set for all Ingress; if set to false, only Ingress with annotation `cf-tunnel-operator.walnuts.dev/cloudflare-tunnel: <CloudflareTunnel namespace>/<CloudflareTunnel name>` annotations.
```

```shell
kubectl apply -f ./cf-tunnel.yaml
```

At this stage, the Cloudflared pod should be up and running, and Cloudflare Tunnel and DNS settings should have been created for all ingresses in the cluster.

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
make setup
tilt up --host 0.0.0.0
```

### Stop Cluster

```shell
make stop
```
