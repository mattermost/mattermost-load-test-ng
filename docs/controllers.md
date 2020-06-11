# Controllers

## Introduction

The `UserController` is the component in charge of driving/controlling a user.

Its implementation is what defines user's behaviour. This includes any
action that the user might do like logging in, posting a message, updating the
profile etc.

## Available Controllers

We currently provide and support the following implementations:

### `SimpleController`

This is a simple version of a controller. It will run a pre-defined (and
configurable) set of actions in a loop.  
It's configurability and granularity make it a good choice to test performance
changes for single API calls.  
This is the recommended controller when the goal of the load-test is to figure
out if some change in the code might have had an impact on performance related
to a specific subset of actions.  

### `SimulController`

This is the most simulative version of a controller. It will try and mimic real
user behaviour.  
It's the recommended version to use when the goal of the
load-test is finding out how many concurrently active users the target instance
supports.  
It can also be used as a way of doing smoke testing around the backend code.  

### `NoopController`

This is a controller that runs the minimum amount of actions needed to connect a user.  
It is used to calculate what is the ideal performance of a Mattermost instance.  
It's sole purpose is to have the user login once, open a WebSocket connection and perform just one request.  
Running this controller will serve as a baseline against which to compare other results.  

### `GenController`

This controller's purpose is to generate data (teams, channels, posts, etc.).  
This is particularly useful when a more realistic starting setup is required.  
Also, it is used to populate an empty database during the init process.  
