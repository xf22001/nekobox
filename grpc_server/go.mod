module grpc_server

go 1.21.5

require (
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.33.0
	libneko v0.0.0
)

replace libneko v0.0.0 => ../libneko

require (
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
)
