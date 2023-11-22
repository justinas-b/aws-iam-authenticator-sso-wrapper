# -*- mode: Python -*-

PROJECT_NAME   = 'aws-iam-authenticator-sso-wrapper'
IMAGE_REGISTRY = 'ttl.sh'
IMAGE_REPO     = 'aws-iam-authenticator-sso-wrapper'

# Building go binary locally
local_resource(
  name = 'build',
  cmd  = 'GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -gcflags "all=-N -l" -o ./build/%s .' % PROJECT_NAME,
  deps = [ # list of file that will trigger rebuild when modified
    './main.go',
    './aws.go',
    './kubernetes.go',
    './type.go'
  ],
)

# Set the default registry where image will be pushed to after build and from where it will be pulled in k8s
default_registry('%s/%s' % (IMAGE_REGISTRY, IMAGE_REPO))

# Use custom Dockerfile for Tilt builds, which only takes locally built binary for live reloading.
dockerfile = '''
    FROM golang:1.19-alpine
    RUN go install github.com/go-delve/delve/cmd/dlv@latest
    COPY ./build/%s /usr/local/bin/%s
    ''' % (PROJECT_NAME, PROJECT_NAME)

# Wrap a docker_build to restart the given entrypoint after a Live Update.
load('ext://restart_process', 'docker_build_with_restart')
docker_build_with_restart(
    ref                 = PROJECT_NAME,
    context             = '.',
    dockerfile_contents = dockerfile,
    entrypoint          = '/go/bin/dlv --listen=0.0.0.0:50100 --api-version=2 --headless=true --only-same-user=false --accept-multiclient --check-go-version=false exec /usr/local/bin/%s -- -dst-namespace=%s -dst-configmap=aws-auth-dst -src-configmap=aws-auth-src -interval=1800 -debug' % (PROJECT_NAME, PROJECT_NAME),
    only                = './build/%s' % PROJECT_NAME, # trigger docker image rebuild only if go binary has been recompiled
    live_update         = [
        # Copy the binary so it gets restarted.
        sync(
          local_path = PROJECT_NAME,
          remote_path = '/usr/local/bin/%s' % PROJECT_NAME,
        ),
    ],

)

# Allow the cluster to avoid problems while having kubectl configured to talk to a remote cluster.
allow_k8s_contexts(k8s_context())

# Load .env file
load('ext://dotenv', 'dotenv')
dotenv()

# Provision kubernetes resources
k8s_yaml('tilt-files/namespace.yaml')
k8s_yaml('tilt-files/configMap.yaml')
k8s_yaml('tilt-files/deployment.yaml')
k8s_yaml('tilt-files/roles.yaml')

# Replace IAM role used on SA with one defined in .env file
objects = read_yaml_stream('tilt-files/serviceAccount.yaml')
for o in objects:
  o['metadata']['annotations']['eks.amazonaws.com/role-arn'] = os.environ['IAM_ROLE']
k8s_yaml(encode_yaml_stream(objects))

# Configure port-forwarding for delve
k8s_resource(
  workload      = PROJECT_NAME,
  port_forwards = ["50100:50100"], # Set up the K8s port-forward to be able to connect to it locally.
  resource_deps = [
    'build',
  ],
)

