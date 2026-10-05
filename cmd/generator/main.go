package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/pipeline"

	"github.com/l-nmch/provider-openfga/config"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		panic("root directory is required to be given as argument")
	}
	rootDir := os.Args[1]
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		panic(fmt.Sprintf("cannot calculate the absolute path with %s", rootDir))
	}
	provider := config.GetProvider()
	pipeline.Run(provider, config.GetProviderNamespaced(), absRootDir)

	// The list of generated Terraform resources, which "make
	// schema-version-diff" reads to report native state schema version changes
	// when the Terraform provider is bumped. Written here so that it cannot
	// drift from the resources actually generated.
	if err := writeGeneratedList(filepath.Join(absRootDir, "config", "generated.lst"), provider); err != nil {
		panic(fmt.Sprintf("cannot write the generated resource list: %s", err))
	}
}

func writeGeneratedList(path string, provider *ujconfig.Provider) error {
	names := make([]string, 0, len(provider.Resources))
	for name := range provider.Resources {
		names = append(names, name)
	}
	sort.Strings(names)
	out, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644) //nolint:gosec // a list of resource names
}
