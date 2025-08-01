package runtimes

import (
	"github.com/sourcenetwork/lens/host-go/engine/module"
	"github.com/sourcenetwork/lens/host-go/runtimes/js"
)

func Default() module.Runtime {
	return js.New()
}
