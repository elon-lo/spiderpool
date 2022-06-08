// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package constant

import (
	"errors"
)

var (
	ErrInternal          = errors.New("internal server error")
	ErrWrongInput        = errors.New("wrong input information")
	ErrNotAllocatablePod = errors.New("not allocatable pod")
	ErrNoAvailablePool   = errors.New("no available IP pool")
	ErrIPUsedOut         = errors.New("all IP used out")
)
