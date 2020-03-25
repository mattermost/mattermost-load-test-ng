# Coordinator

The `coordinator` is the component in charge of managing a cluster of load-test
agents and figure out what's the maximum amount of concurrently active users
the target instance can support.

This is achieved through the use of a simple feedback loop.  
When a load-test starts the coordinator will gradually increase the number of
active users. At the same time it will monitor performance by querying a
Prometheus server collecting metrics from the target instance.
If signs of performance degradation are detected the coordinator will start
decreasing the number of active users. This process continues until an equilibrium point is found.

## The feedback loop

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
