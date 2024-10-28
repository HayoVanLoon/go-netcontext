GO := $(shell which go)

OUT := go-genproto

check-googleapis:
ifndef PROTO_GOOGLEAPIS
	$(error Missing environment variable PROTO_GOOGLEAPIS)
endif

protoc: check-googleapis
	rm -rf "$(OUT)"
	./compile.sh \
		-i proto \
		--no-gateway \
		--no-mod \
		-o "$(OUT)"

call-success:
	curl -v 'http://localhost:8080/deadline?todo=9&timeout=5'

call-maybe:
	curl -v 'http://localhost:8080/deadline?todo=13&timeout=5'

call-fail:
	curl -v 'http://localhost:8080/deadline?todo=21&timeout=5'


install:
	$(GO) install github.com/golang/protobuf/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
