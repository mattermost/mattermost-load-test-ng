/*
Package [plugins] exposes a series of interfaces and functions for plugins to
register logic into the different controllers of the load-test tool:
 1. [Controller] is the core of the package: an interface that plugins need to
    implement so that they can be registered and their actions executed during
    the controller's main loop. It is extended by [SimulController] and
    [GenController].
 2. [RegisterController] is the function that plugins need to call, usually inside
    their package's init function, to register their implementations of the
    [Controller] interface.
 3. [HookType] and the HookPayloadXYZ structs are the core of the hooks:
    additional logic for the plugins to inject code into regular actions owned
    by the load-test tool.
*/
package plugins

import (
	"sync"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// Controller is the generic interface defining the methods to implement by
// Mattermost plugins to be injected into different controllers
type Controller interface {
	// PluginId must return the identifier of the plugin.
	PluginId() string

	// MinServerVersion must return the minimum Matermost server version needed
	// for these plugin's actions to work.
	MinServerVersion() semver.Version

	// Actions must return a list of all the standalone actions implemented by the
	// plugin.
	Actions() []PluginAction

	// ClearUserData must clear all user's data in the plugin's store.
	ClearUserData()
}

// SimulController is an interface that extends [Controller] with the functions needed
// only by the SimulController logic.
type SimulController interface {
	Controller

	// RunHook must run the logic corresponding to the [HookType]. It receives
	// the user and a generic payload, that should be converted to the proper
	// type ([HookPayloadLogin], [HookPayloadSwitchTeam],
	// [HookPayloadSwitchChannel]...) to access the data.
	RunHook(hookType HookType, u user.User, payload any) error
}

// GenController is an interface that extends [Controller] with the functions needed
// only by the GenController logic.
type GenController interface {
	Controller

	// Done must return whether the generative controller has finished
	// generating the data it is configured to.
	Done() bool
}

// PluginAction contains all the information for a plugin's action to be
// properly registered and ran.
type PluginAction struct {
	// Name is a unique string identifying the action. It must be unique among
	// all the actions provided by a single plugin.
	Name string
	// Run is the function that implements the logic in the plugin's action.
	Run control.UserAction
	// Frequency is the relative frequency with which the simulation will pick
	// this action to run.
	Frequency float64
}

// The global lock to protect access to [registeredPluginsByType]
var pluginsLock sync.RWMutex

// controllerFunc is the type of a function returning a [Controller].
// It is used by plugins to register themselves against the controllers.
type controllerFunc = func() Controller

// The global map of registerd plugins, mapping each [ControllerType] to a list
// of functions that generate a [SimulController].
var registeredPluginsByType = map[ControllerType][]controllerFunc{}

// ControllerType is the type of the load-test controller a plugin [Controller]
// should be injected to.
type ControllerType string

const (
	// TypeSimulController is the type of [Controller]s that should be injected into
	// the [simulcontroller.SimulController] controller.
	TypeSimulController ControllerType = "simulcontroller"

	// TypeGenController is the type of [Controller]s that should be injected into
	// the [gencontroller.GenController] controller.
	TypeGenController ControllerType = "gencontroller"
)

// RegisterController registers the function f, which is called
// whenever a new instance of the plugin controller is spawn to be used by the
// load-test controllers.
// The provided function f must return a type implementing the controller
// specified by controllerType.
func RegisterController(controllerType ControllerType, f controllerFunc) {
	pluginsLock.Lock()
	defer pluginsLock.Unlock()

	if registeredPluginsByType[controllerType] == nil {
		registeredPluginsByType[controllerType] = []controllerFunc{}
	}
	registeredPluginsByType[controllerType] = append(registeredPluginsByType[controllerType], f)
}
