package plugins

// GeneratePluginControllers loops over all the registered plugins of type
// [controllerType], calling the provided function [f] with each of them.
//
// This function is exposed for other packages under mattermost-load-test-ng to
// use, it should not be used by external plugins.
func GeneratePluginControllers(controllerType ControllerType, f func(p Plugin)) {
	pluginsLock.RLock()
	defer pluginsLock.RUnlock()

	for _, genPlugin := range registeredPluginsByType[controllerType] {
		f(genPlugin())
	}
}
