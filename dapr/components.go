package dapr

import (
	"github.com/dapr/components-contrib/state/redis"
	"github.com/taction/wit-dapr/pkg/imports/host-state/registery"
)

func init() {
	registery.DefaultRegistry.RegisterComponent(redis.NewRedisStateStore, "redis")
}
