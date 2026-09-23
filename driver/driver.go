package driver

import (
	"go.gh.ink/notifyutils/internal/state"
	"go.gh.ink/notifyutils/model"
)

func Register(name string, driver model.Driver) {
	state.Drivers[name] = driver
}
