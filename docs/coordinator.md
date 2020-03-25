# Coordinator

## Introduction

The `coordinator` is the component in charge of managing a cluster of load-test
agents and figure out what's the maximum amount of concurrently active users
the target Mattermost instance can support. This is achieved through the use of a simple feedback loop.

When a load-test starts, the `coordinator` will begin to gradually increase the number of
active users. At the same time it will monitor performance by querying a
[Prometheus](https://prometheus.io/docs/introduction/overview/) server collecting metrics from the target instance.

When signs of performance degradation are detected, the `coordinator` will start
decreasing the number of active users. When overall performance goes back to an
accepted value (results of queries fall under the configured thresholds) the `coordinator` will start once again to increase the amount
of active users for the target instance.

This process will continue until an equilibrium point is found which will
indicate the estimated maximum number of supported users.

## The feedback loop

```

                       +------------------------------------------+
              +--------v--------+                                 |
              |                 |                                 |
      +-------+   coordinator   +---------+                       |
      |       |                 |         |                       |
      |       +--------+--------+         |                       |
      |                |                  |                       |
      |                |                  |                       |
+-----v-------+  +-----v-------+  +-------v-----+                 |
|  load-test  |  |  load-test  |  |  load-test  |                 |
|    agent    |  |    agent    |  |    agent    |                 |
+-----+-------+  +------+------+  +------+------+                 |
      |                 |                |                        |
      |                 |                |                        |
      |          +------v------+         |                        |
      |          |  Mattermost |         |                        |
      +--------->+   instance  +<--------+                        |
                 +------+------+                                  |
                        |                                         |
                        v                                         |
                 +------+-------+                                 |
                 |              |                                 |
                 |  Prometheus  +---------------------------------+
                 |              |
                 +--------------+
```
