// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/SamaNetwork/SamaVM/vm"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/vms"
)

var _ vms.Factory = &Factory{}

type Factory struct{}

func (f *Factory) New(*snow.Context) (interface{}, error) { return &vm.VM{}, nil }
