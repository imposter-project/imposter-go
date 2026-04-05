package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// methodDescriptors holds the input and output message descriptors for a gRPC method.
type methodDescriptors struct {
	input  protoreflect.MessageDescriptor
	output protoreflect.MessageDescriptor
}

// parseProtoFiles parses the given .proto files and returns a map of
// gRPC path (e.g. "/store.PetStore/GetPet") to method descriptors.
func parseProtoFiles(configDir string, protoFiles []string, logger hclog.Logger) (map[string]*methodDescriptors, error) {
	// Build absolute paths for proto files
	absPaths := make([]string, len(protoFiles))
	for i, f := range protoFiles {
		if filepath.IsAbs(f) {
			absPaths[i] = f
		} else {
			absPaths[i] = filepath.Join(configDir, f)
		}
	}

	// Verify proto files exist
	for _, p := range absPaths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("proto file not found: %s", p)
		}
	}

	// Configure the compiler with the config directory as an import path
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: []string{configDir},
		},
		Reporter: reporter.NewReporter(
			func(err reporter.ErrorWithPos) error {
				logger.Error("proto compile error", "error", err)
				return err
			},
			func(err reporter.ErrorWithPos) {
				logger.Warn("proto compile warning", "warning", err)
			},
		),
	}

	// Compile using the relative file names (resolved against import paths)
	compiled, err := compiler.Compile(context.Background(), protoFiles...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile proto files: %w", err)
	}

	// Build the method lookup map
	methods := make(map[string]*methodDescriptors)
	for _, fd := range compiled {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			svcFullName := string(svc.FullName())
			svcMethods := svc.Methods()
			for j := 0; j < svcMethods.Len(); j++ {
				m := svcMethods.Get(j)
				grpcPath := fmt.Sprintf("/%s/%s", svcFullName, m.Name())
				methods[grpcPath] = &methodDescriptors{
					input:  m.Input(),
					output: m.Output(),
				}
				logger.Debug("registered gRPC method", "path", grpcPath)
			}
		}
	}

	if len(methods) == 0 {
		logger.Warn("no gRPC methods found in proto files")
	}
	return methods, nil
}
