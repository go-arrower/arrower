//nolint:govet // fieldalignment less important than readability.
package hooks

import "strings"

// Register adds the given hook to the stack.
func Register(hook Hook) {
	if hook.Name == "" {
		hook.Name = "<unknown>"
	}

	if hook.OnConfigLoaded == nil {
		hook.OnConfigLoaded = func(*RunConfig) {}
	}

	if hook.OnStart == nil {
		hook.OnStart = func() {}
	}

	if hook.OnChanged == nil {
		hook.OnChanged = func(string) {}
	}

	if hook.OnShutdown == nil {
		hook.OnShutdown = func() {}
	}

	loadedHooks = append(loadedHooks, hook)
}

// RunConfig is the configuration used for the `arrower run` cli command.
type RunConfig struct {
	// Port is for the webserver arrower starts and where it offers additional
	// information and features regarding the status of the project in development.
	// Arrower apps check in to this port as well, e.g. for hot reload signals.
	Port int

	// WatchPath is the directory that arrower watches all file changes from.
	WatchPath string
}

type Hook struct {
	Name           string
	OnConfigLoaded func(c *RunConfig)
	OnStart        func()
	OnChanged      func(file string)
	OnShutdown     func()
}

type Hooks []Hook

// NamesFmt returns a printable list of the hook names, like:
// hook 0, hook 1, hook 2.
func (h Hooks) NamesFmt() string {
	names := make([]string, len(h))

	for i, hook := range h {
		names[i] = hook.Name
	}

	return strings.Join(names, ", ")
}

func (h Hooks) OnConfigLoaded(c *RunConfig) {
	for _, hook := range h {
		hook.OnConfigLoaded(c)
	}
}

func (h Hooks) OnStart() {
	for _, hook := range h {
		hook.OnStart()
	}
}

func (h Hooks) OnChanged(file string) {
	for _, hook := range h {
		hook.OnChanged(file)
	}
}

func (h Hooks) OnShutdown() {
	for _, hook := range h {
		hook.OnShutdown()
	}
}
