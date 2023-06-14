# Documentation

## Where do I start?

The documentation here is comprehensive, so it may be hard for someone new to know where to start reading to understand how the tool works or how it is architected. The best starting point depends on what your end goal is:

- If you are interested in using the tool to run some tests, there are a few documents you need to read, roughly in the following order:
    - [How-to guide](load-test-how-to-use.md): this guide is a great overview of how things work, as well as tips and tricks that will make your life easier. It also links to multiple other documents, which you can follow to broaden your understanding. Read it completely first before running any tests!
    - [How to run a load-test locally](local_loadtest.md): a specific guide to start running load-tests locally in your computer, no AWS needed.
    - [How to run a load-test in Terraform](terraform_loadtest.md): a more advanced guide to deploy a cluster to AWS to run a more powerful load-test.
    - [Generate and compare reports](compare.md): once the test is executed, here's how you can generate a report you can use for analyzing the results, as well as for comparing it against a second test to generate a diff view and some useful graphs.
    - [FAQ](faq.md): whenever you find yourself wondering something about the tool, check here first in case other people have already asked.
- If you are more interested in learning how the tool is architected, or how to improve it, then you should start here:
    - [Architecture](loadtest_system.md): a birds-eye view of what the different components are and how they connect to each other.
    - [Implementation](implementation.md): an overview, both high- and low-level, of how the different components are implemented.
    - [Coverage](coverage.md): if you have implemented a new feature in the server, here's a guide on how to modify the simulcontroller to add coverage for it.
    - [Developer's workflow](developing.md): quick recipes to get you started in developing for the tool.
    
## Reference documentation

### Configuration

Users of the tool can configure pretty much everything: the behaviour of the simulated users, how to set up the cluster, the metrics and thresholds for how the coordinator measures the stability of the system, and even AWS-specific settings, to name a few examples.

This is great for accommodating all of the use-cases we need to cover, but it also makes the configuration quite complex. To try to alleviate this pain, there's extensive documentation on each and every configuration knob you can tweak (and if you see something's missing, please feel free to open an issue or a PR). Here's a list of all the documents explaining each of the files in the `config/` directory:

- [Generic configuration](config/config.md): Settings to configure what controller to use (see below for more info), how the controllers will connect to your Mattermost server, how many users each controller has and how active they are, as well as the initial data you want to create.
- Controllers configuration: settings for each different controller implemented in the tool:
    - [simulative controller](config/simulcontroller.md): the perfect controller for simulating a real user, with coverage of the most frequent actions and fine-tuned probabilities to mimic an actual server.
    - [generative controller](config/gencontroller.md): the controller you'll use whenever you need to generate lots of data for later use. The behaviour of the users in this controller is not realistic (hopefully real users don't write 10+ posts per second), but it's optimized for generating data with speed and performance in mind.
    - [simple controller](config/simplecontroller.md): if you need to run a specific set of actions in a loop, and you want to easily configure which ones and with which frequency, then this is your controller.
- [Deployer configuration](config/deployer.md): Settings to control the cluster deployed to AWS through Terraform, as well as AWS-specific configurations you may need to tweak to match you usual AWS workflow.
- [Coordinator configuration](config/coordinator.md): Settings to tweak how the coordinator behaves, such as the maximum number of users simulated or the configuration for the type of test to run (whether bounded or unbounded, and in case of unbounded, how it is controlled).
- [Comparison configuration](config/comparison.md): Settings to control the comparison process. See the [Advanced workflows](#advanced-workflows) for more information.

### Components

Specific documentation defining the main components and how they interact with each other:

- [Controllers](controllers.md): what a controller is, as well as a list of the different implementations included.
- [Coordinator](coordinator.md): a definition of the coordinator component and how it works.

## Advanced Workflows

Once you have familiarized yourself with the tool, and after you have successfully run at least a couple of tests, there are a few advanced guides that may be useful for you:

- [Running an automated load-test comparison](comparison.md): a workflow specifically designed for when you need to compare two different versions of Mattermost while maintaining the rest of the variables fixed. This is what the Server team at Mattermost uses for the monthly release performance comparisons.
- [Generating data](generating-data.md): for larger load-tests, you'll need larger datasets. This guide describes how you can use the gencontroller to create an arbitrary number of teams, channels, posts, reactions... to use as the starting point for future tests.

