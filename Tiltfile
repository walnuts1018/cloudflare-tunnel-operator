load('ext://restart_process', 'docker_build_with_restart')

DOCKERFILE = '''FROM golang:alpine
WORKDIR /
COPY ./bin/manager /
CMD ["/manager"]
'''

def manifests():
    return 'controller-gen crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases;'

def go_generate():
    return 'go generate ./...;'

def generate():
    return 'controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./...";'

def binary():
    return 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/manager cmd/main.go;'

def ingress_nginx():
    return 'kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml;'

# Generate manifests and go files
local_resource('make manifests', manifests(), deps=["api", "internal", "hooks"], ignore=['*/*/zz_generated.deepcopy.go'])
local_resource('make generate', go_generate() + generate(), deps=["api", "hooks"], ignore=['*/*/zz_generated.deepcopy.go'])

# Deploy CRD
local_resource(
    'CRD', manifests() + 'kustomize build config/crd | kubectl apply -f -', deps=["api"],
    ignore=['*/*/zz_generated.deepcopy.go'])

# Deploy manager
watch_file('./config/')
# k8s_yaml('./config/dev/namespace.yaml')
k8s_yaml(kustomize('./config/dev'))

local_resource(
    'Watch & Compile', generate() + binary(), deps=['internal', 'api', 'cmd/main.go'],
    ignore=['*/*/zz_generated.deepcopy.go'])

docker_build_with_restart(
    'controller:latest', '.',
    dockerfile_contents=DOCKERFILE,
    entrypoint=['/manager'],
    only=['./bin/manager'],
    live_update=[
        sync('./bin/manager', '/manager'),
    ]
)

local_resource('Sample Dependencies', ingress_nginx())

local_resource(
    'Sample', 'kustomize build ./config/samples | kubectl apply -f -',
    deps=["./config/samples"])
