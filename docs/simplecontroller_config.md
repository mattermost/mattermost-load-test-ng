# SimpleControllerConfiguration

## Rate

*float64*

Rate is the idle time coefficient for user actions that will be performed sequentially.

- A rate < 1.0 will run actions at a faster pace.
- A rate == 1.0 will run actions at the default pace.
- A rate > 1.0 will run actions at a slower pace.

This value is multiplied with [WaitAfterMs](#WaitAfterMs).

## Actions

*[]ActionDefinition*

Actions are the user action definitions that will be run by the SimpleController.

### ActionDefinition

#### ActionId

*string*

Action name which is mapped to SimpleController's actions. Available actions can be found [here](https://github.com/mattermost/mattermost-load-test-ng/blob/master/loadtest/control/simplecontroller/controller.go#L137).

#### WaitAfterMs

*integer*

Wait time in milliseconds after the action is performed.

#### RunFrequency

*integer*

The value of how often the action will be performed. The higher the value the lesser possibility to run the action.
