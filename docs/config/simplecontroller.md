# SimpleController Configuration

## Actions

*[]ActionDefinition*

Actions are the user action definitions that will be run by the SimpleController.

### ActionDefinition

#### ActionId

*string*

Action name which is mapped to SimpleController's actions. Available actions can be found [here](https://github.com/mattermost/mattermost-load-test-ng/blob/master/loadtest/control/simplecontroller/controller.go#L137).

#### WaitAfterMs

*int*

Wait time in milliseconds after the action is performed.

#### RunPeriod

*int*

The value of how often the action will be performed. The higher the value the lesser possibility to run the action.

- A RunPeriod > 0 is expected to run the action.
- A RunPeriod == 0 will cause the action to be skipped.
- A RunPeriod < 0 will cause an error while validating the configuration.
