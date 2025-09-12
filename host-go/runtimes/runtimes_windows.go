//go:build windows && !js && !cshared

package runtimes

import (
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/runtimes/wazero"
)

func Default() module.Runtime {
	return wazero.New()
}
