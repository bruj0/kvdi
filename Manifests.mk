## Cluster manifests

ifeq ($(strip $(DOCKER_HOST)),)
export API_SERVER_ADDRESS=127.0.0.1
else
export API_SERVER_ADDRESS=$(DOCKER_HOST)
endif


define KIND_CLUSTER_MANIFEST
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: "${API_SERVER_ADDRESS}"
nodes:
- role: control-plane
- role: worker
  extraMounts:
  - hostPath: /dev/shm
    containerPath: /dev/shm
  - hostPath: /dev/snd
    containerPath: /dev/snd
endef

define METALLB_CONFIG
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - 172.17.255.1-172.17.255.250
endef


export KIND_CLUSTER_MANIFEST
export METALLB_CONFIG
##
