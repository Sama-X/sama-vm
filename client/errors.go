// Copyright (C) 2022-2023, Sama , Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import "errors"

var ErrIntegrityFailure = errors.New("received file that does not match hash")
