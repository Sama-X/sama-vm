// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tree

import (
	"errors"
)

var (
	ErrEmpty   = errors.New("file is empty")
	ErrMissing = errors.New("required file is missing")
)
