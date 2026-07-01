package main

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/types/plugin"
)

type handler struct{}

func (h *handler) GenerateCode(_ thrift.Context, req *plugin.GenerateCodeRequest) (*plugin.GenerateCodeResponse, error) {
	var buf bytes.Buffer

	// Namespaces — sorted by key for determinism
	nsKeys := make([]string, 0, len(req.Program.Namespaces))
	for k := range req.Program.Namespaces {
		nsKeys = append(nsKeys, k)
	}

	sort.Strings(nsKeys)

	for _, k := range nsKeys {
		fmt.Fprintf(&buf, "namespace %s: %s\n", k, req.Program.Namespaces[k])
	}

	// Structs — sorted by name
	structNames := make([]string, 0, len(req.Program.Structs))
	for name := range req.Program.Structs {
		structNames = append(structNames, name)
	}

	sort.Strings(structNames)

	fmt.Fprintf(&buf, "structs: %s\n", strings.Join(structNames, ", "))

	// Services — sorted by name
	serviceNames := make([]string, 0, len(req.Program.Services))
	for name := range req.Program.Services {
		serviceNames = append(serviceNames, name)
	}

	sort.Strings(serviceNames)

	fmt.Fprintf(&buf, "services: %s\n", strings.Join(serviceNames, ", "))

	return &plugin.GenerateCodeResponse{
		Files: map[string][]byte{
			"echo.txt": buf.Bytes(),
		},
	}, nil
}

func main() {
	plugin.Execute(context.Background(), &handler{})
}
