module github.com/antigravity/mono

go 1.22

// This is the root workspace go.mod for shared tooling and proto gen only.
// Each service has its own go.mod for independent versioning.

require (
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.34.0
)
