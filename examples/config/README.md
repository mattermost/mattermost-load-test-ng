# Configuration samples

This directory contains sets of configuration templates that we use in different scenarios. Some fields are hard-coded to the values we use in our day-to-day processes (e.g. the path to the SSH keys), and others are marked as `#TBD` because they may change from run to run (e.g. the URLs to download Mattermost from). In any case, these sets can serve as starter packs for other, different workflows. For now, we have:
- [Release testing](./release): configuration used when testing a new release of the load-test tool.
- [Performance comparison](./perfcomp): configuration used for regression testing of new Mattermost releases. The results of these runs can be found in the [`performance-reports` repository](https://github.com/mattermost/performance-reports/tree/main/performance-comparisons).
