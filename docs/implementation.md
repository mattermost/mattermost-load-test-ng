## Code Structure / Packages

- cmd
	 - loadtest
- loadtest
	- control
		- simplecontroller
	- user
		- userentity
	- store
		- memstore

## How it works (high level view)

The main idea behind this refactor/rewrite is to move to a more user centered model. A user is made of two parts: its state (what it knows) and its behaviour (what it does).   
It's important we keep the two aspects completely separated in order to avoid unnecessary complexities.

By defining a user this way we make it independent and should be easy to spawn as many entities as we need and obtain a higher quality simulation.

Under this model a user will be running loops of this kind:

Sign up --> Log in --> Do stuff --> ... --> Logout --> Log in --> Do stuff --> ...

The implementation tries to keep everything abstracted enough to make its components as loosely coupled as possible and at the same time allow for good extensibility.

## Implementation Details (low level view)

### Interfaces

- `UserController`
	this is the actor that's in charge of user's behaviour. It has *almost* readonly access to the user's state. This is where the logic of "what/when the user does" goes. You can think about it as the user's *mind*.
I envision multiple implementations going from a very simple, deterministic one (included) to something more realistic (stochastic approach) like we currently have in the master branch.

- `User` 
   this exposes basic user's actions (signup/login/logout/post, etc) , handles API client and deals with state management.
   Made as an interface to hide implementation details from outside while keeping state handling confined.
   
- `UserStore`
  this is the user's state (*brain*), think of this as what redux is for webapp (1000 times simplified). Should expose basic *readonly* access to the user's state (user/channels/teams, etc). It is used by the `UserController` implementation.

- `MutableUserStore`
  this is a supertype of `UserStore` that adds the write functionality. It is used by the `User` implementation to manage the internal user state.

### Concrete Types

- `LoadTester`
   this is the main point of entry . It's currently a convenient singleton that initializes and operates the controllers and handles goroutines synchronization.
- `SimpleController`
   simple (dumb) implementation of `UserController`. It will run user actions sequentially. Currently executes signup/login/logout loops.
- `UserEntity`
   This is the type that implements `User`. It holds API and WS clients and has full access to the underlying store. This is where user's state management is implemented.
- `MemStore`
   A very basic *in memory* state implementation of `MutableUserStore` mainly consisting of maps of structs the user needs to operate.
