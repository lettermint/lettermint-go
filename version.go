package lettermint

import (
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
)

// modulePath is the import path of this module. A new major version changes it.
const modulePath = "github.com/lettermint/lettermint-go/v3"

var version = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	return moduleVersion(info)
})

// moduleVersion finds this module's version in the build information of the
// program: the release tag (v3.1.0) when the SDK is a dependency, or "dev"
// for a local checkout, a replace directive or a test of this module.
func moduleVersion(info *debug.BuildInfo) string {
	modules := append([]*debug.Module{&info.Main}, info.Deps...)
	for _, module := range modules {
		if module == nil || module.Path != modulePath {
			continue
		}
		if module.Replace != nil {
			module = module.Replace
		}
		if module.Version == "" || module.Version == "(devel)" {
			return "dev"
		}
		return strings.TrimPrefix(module.Version, "v")
	}
	return "dev"
}

// Version returns the SDK version, such as "3.0.0", read from the program's
// build information (the module version Go selected). It is "dev" when the
// SDK is built from a local checkout.
func Version() string {
	return version()
}

// userAgent is sent with every request: lettermint-go/3.0.0 (go1.24.1).
func userAgent() string {
	return "lettermint-go/" + Version() + " (" + runtime.Version() + ")"
}
