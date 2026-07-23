package opencode

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	ProvideService,
)
