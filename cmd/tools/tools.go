//go:build tools

package tools

import (
	// protoc-gen-go-grpc generates Go gRPC service code from proto definitions.
	// Pinned via go.mod to ensure consistent protoc plugin versions.
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"

	// protoc-gen-go generates Go protobuf message code from proto definitions.
	// Pinned via go.mod to ensure consistent protoc plugin versions.
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
