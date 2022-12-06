# Action frequency

When adding coverage for a new action to `SimulController`, you'll need to define its relative frequency with respect to other actions: is the new action as frequent as, let's say, creating a post, or is it more similar to updating the custom status? This number is used by the controller to choose which action to perform, so one with a value of ten is two times more likely to happen than one with a value of five. But how does one come up with such a number?

There's actually not a 100% right way to do it, since ideally we would like to model all types of Mattermost servers out there, and they all differ from each other. Even the same server may show different frequencies now to the ones seen one year ago. However, we can try our best and at least be internally consistent, so the frequencies of all new actions should be calculated following this guide.

Go through the following section for a short bullet list of what's needed, or read the Detailed guide section if you want a more thorough explanation.

## Summary

In short, this whole process can be summarized as follows:

1. Log in to [grafana.internal.mattermost.com](https://grafana.internal.mattermost.com).
2. Choose the two-hour window with the most traffic for the past seven days.
3. Run the following query in that window:
```promql
sum(rate(mattermost_api_time_count{handler=~"the-name-of-your-handler"}[2h]))
/
sum(rate(mattermost_api_time_count{handler=~"createPost"}[2h]))
```
4. Get the last value at the very end of the window.
5. Multiply that value by 1.5 (the frequency of `createPost`). That's your frequency!

## Detailed guide

The frequencies assigned to actions in this controller are extrapolated from real frequencies seen in the Community server, so when adding a new one, we need to look at data from [community.mattermost.com](https://community.mattermost.com) to decide which frequency it'll have. This guide assumes that the feature was deployed to Community at least seven days prior. If you're adding coverage to the load-test before that, you'll have to simply *guess* how your action relates to the other ones, since there's no actual data out there to base your decision on. Once the feature is merged and a week has passed, you can follow this guide and hone that number in a second PR.

The general idea is to analyze metrics data in Grafana and compare the new action against a well-known one during a busy period of time. Let's dissect all that.

### Logging into Grafana

The Community server has a Prometheus and Grafana system set up to analyze metrics. All Mattermost developers should have access. To check it:

1. Make sure you're connected to the internal VPN.
2. Navigate to [grafana.internal.mattermost.com](https://grafana.internal.mattermost.com) and sign in with SAML.

If you have troubles logging in, ask the Security Team for access.

### Choose a busy period of time

All action frequencies should be calculated during a time where the server is under high traffic. To find out such a time, go to the [Number of Connected Devices (Websocket Connections)](https://grafana.internal.mattermost.com/d/000000011/mattermost-performance-monitoring?orgId=4&refresh=5s&viewPanel=6&from=now-7d&to=now) panel and check the concurrent number of connected users at any point during the last seven days. Usually, Wednesdays and Thursdays during US/Canada mornings are the busiest hours, but it may change. Note down the two hours where you see the most traffic: you can simply click and drag in the graph to select the range of time, which will select the start and end time in the time selector.

![Number of connected devices](https://community.mattermost.com/files/5dz63kgi8id3mfdik6jy9zhddy/public?h=9tK_dYBYiKGyseJvrD3nxLf9EhGXxTcJThFGreJ0LMo)

In this case, the time frame with the most traffics was last Thursday, December 01, between 16:00 UTC and 18:00 UTC, where the system consistently supported between 1.8K and 2K users.

![Number of connected devices at window with most traffic](https://community.mattermost.com/files/jd5zupo3qp8tjxamfiotiqfgca/public?h=6VUIEGmLJmMf60ioL5v5vNfqwgGQMDZ8l9e1rshA2G4)

### Comparing actions

Now's the time to do the actual comparison. First:

1. Navigate to [the Explore section](https://grafana.internal.mattermost.com/explore), where you can play with custom Prometheus queries.
2. Make sure that you have `Prometheus Community` selected in the top dropdown menu.
3. Select the times you noted in the step before to focus only on the busiest time slot of the week.

![Selectors](https://community.mattermost.com/files/x3w8u54k8bfmjcxxrhkgm1h4ew/public?h=e0luDLAuSCUwKmQyzgHNqYD4w8Tt0wf7vWs0R0m53Fo)

We can now start writing some Prometheus queries. Let's say that the frequency of the new action can be represented by a new API endpoint whose handler is `upsertDraft`. Then, to compare it with a well-known action such as create post, we can get their ratio:

```promql
sum(rate(mattermost_api_time_count{handler=~"upsertDraft"}[2h]))
/
sum(rate(mattermost_api_time_count{handler=~"createPost"}[2h]))
```

Let's understand this a bit better, focusing first in the first line:

1. `mattermost_api_time_count` is the metric used by the system to sample the number of times we hit each API. It is just a counter of how many times each endpoint is hit at each point in time.
2. This metric has several labels, such as `method` to differentiate between different HTTP verbs, `status_code` to filter by the status code returned by the server or `handler`, which is the one used here: the name of the API handler we're interested in. We can filter by different labels with the special `{}` syntax. Then, `mattermost_api_time_count{handler=~"upsertDraft"}` filters the values of this metric whose handler matches (`=~`) the regex passed. In this case, the regex is simply a literal string, `upsertDraft`.
3. Now, the previous filter is still a plain counter (it tells us that at time `T`, the `upsertDraft` endpoint was hit `n` times), but we're more interested in the rate at which the endpoints are hit; i.e., how many requests per second we're serving for this specific API. For that, we need to first convert this counter to [a range vector](https://prometheus.io/docs/prometheus/latest/querying/basics/#range-vector-selectors) with the syntax `[2h]`: this gets, at time `T`, a vector of all values between `T` minus two hours and `T`. In a normal query, we're usually more interested in shorter windows, such as `[5m]`, to see how the rate changes in real time, but we're now trying to get a more general view of the whole time window we selected before.
4. The last part of the query (actually, the first syntax-wise), is getting the actual rate with [the `rate` function](https://prometheus.io/docs/prometheus/latest/querying/functions/#rate), which simply returns the per-second average rate at each point in time, taking into account the vector of values we have. In this case, it gets the per-second average rate of the last two hours at each point in time. We finally apply [the `sum` operator](https://prometheus.io/docs/prometheus/latest/querying/operators/#aggregation-operators), which simply aggregates all the labels to get a single value per point in time.

Now that we understand the first line, which gets us the average rate of requests to `upsertDraft` in a two-hour window, we can understand the whole thing. We do exactly the same with `createPost`, so we get the rate to `createPost` in the same two-hour window, and then we divide one between the other. What does that mean? What we obtain is the ratio of rates between these two endpoints; i.e., how they relate to one another: If at a point in time we get that this ratio is `2.0`, that means that the server received two times more requests for `upsertDraft` than for `createPost` in the previous two-hour window. If the value is `0.5`, that means that it's `createPost` the endpoint that is hit double the times than `upsertDraft`. If we draw this line in our time range, we get something like the following:

![Ratio graph](https://community.mattermost.com/files/nrr9zp4a87rp9x8dx66eej961e/public?h=vA0pV04TBVn7mS3wLCzvLxl-_SANGS6tx4i4hOJO5fs)

Now we know, for each point in time between 16:00 and 18:00, the ratio between the rates of `upsertDraft` and `createPost`. But remember that we chose a two-hour window with the `[2h]` range vector selector. So if we look at the time 16:30, we're actually considering data from 14:30 to 16:30. We're mostly interested in the 16:00-18:00 window, so we are actually interested in a single point from this line: the very last one.

We can inspect it by hovering over the last part of the line or by scrolling down and looking at the table below the graph, which conveniently lists the very last value, at 18:00 UTC. In this case, `0.452`.

![Ratio graph - value](https://community.mattermost.com/files/4rnwfgaq7idc9rtm45rye7f44w/public?h=UjBdpoIYymgbjLzgCHA6HIFofDBlIznIMEqpHia0GC0)

Given this ratio, and as the frequency of the `createPost` action is [already configured to be 1.5](https://github.com/mattermost/mattermost-load-test-ng/blob/master/loadtest/control/simulcontroller/controller.go#L125-L128), we can simply multiply these two numbers to get the frequency of the new action:

```
1.5 * 0.452 = 0.678
```

When adding our new action, the `frequency` should be set to `0.678`. Done!

