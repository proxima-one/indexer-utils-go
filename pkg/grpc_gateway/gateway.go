package grpc_gateway

import (
	"context"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"io/fs"
	"mime"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// getOpenAPIHandler serves an OpenAPI UI.
// Adapted from https://github.com/philips/grpc-gateway-example/blob/a269bcb5931ca92be0ceae6130ac27ae89582ecc/cmd/serve.go#L63
func getOpenAPIHandler() http.Handler {
	err := mime.AddExtensionType(".svg", "image/svg+xml")
	if err != nil {
		return nil
	}
	// Use subdirectory in embedded files
	subFS, err := fs.Sub(OpenAPI, "OpenAPI")
	if err != nil {
		panic("couldn't create sub filesystem: " + err.Error())
	}
	return http.FileServer(http.FS(subFS))
}

func getProtoFileHandler(folder fs.FS) http.Handler {
	err := mime.AddExtensionType(".svg", "image/svg+xml")
	if err != nil {
		return nil
	}
	return http.FileServer(http.FS(folder))
}

// Run runs the gRPC-Gateway, dialling the provided address.
func Run(ctx context.Context, grpcAddress string, httpPort int, protoFileFolder fs.FS, protoFileName string,
	registerServiceHandler func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error) error {

	conn, err := grpc.DialContext(
		ctx,
		grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	gwmux := runtime.NewServeMux()
	err = registerServiceHandler(ctx, gwmux, conn)
	if err != nil {
		return fmt.Errorf("failed to register openapi: %w", err)
	}

	openAPIHandler := getOpenAPIHandler()
	protoFileHandler := getProtoFileHandler(protoFileFolder)

	gatewayAddr := fmt.Sprintf("0.0.0.0:%d", httpPort)
	gwServer := &http.Server{
		Addr: gatewayAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") {
				gwmux.ServeHTTP(w, r)
			} else if r.URL.Path == "/proto/schema.json" {
				r.URL.Path = "/" + protoFileName
				protoFileHandler.ServeHTTP(w, r)
			} else {
				openAPIHandler.ServeHTTP(w, r)
			}

		}),
	}

	return fmt.Errorf("serving gRPC-Gateway server error: %w", gwServer.ListenAndServe())
}
