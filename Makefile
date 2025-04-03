# Some rules are copied from https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/blob/main/Makefile

.PHONY: proto-tools
proto-tools:
	cd ./pdata/proto/tools; \
	cat tools.go | grep "_" | awk -F'"' '{print $$2}' | xargs -tI % go install %
	

OPENTELEMETRY_PROTO_VERSION=v1.5.0
.PHONY: submodule-update
submodule-update:
	git submodule update -i
	cd ./pdata/opentelemetry-proto; \
	git checkout $(OPENTELEMETRY_PROTO_VERSION)
	# TODO: consider sparse checkout