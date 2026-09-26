# GenController Configuration

## NumTeams

*int64*

The number of teams to be created.

## NumChannelsDM

*int64*

The number of DM channels to be created.

## NumChannelsGM

*int64*

The number of GM channels to be created.

## NumChannelsPrivate

*int64*

The number of private channels to be created.

## NumChannelsPublic

*int64*

The number of public channels to be created.

## NumCPAFields

*int64*

The number of Custom Profile Attribute fields to be created.

## NumPosts

*int64*

The number of posts to be created.

## NumReactions

*int64*

The number of reactions to be created.

## NumPostReminders

*int64*

The number of post reminders to be created.

## NumScheduledPosts

*int64*

The non-negative global target number of scheduled posts to be created. The default is `0`. A non-zero target requires Mattermost Server 10.3 or later, a valid license, scheduled posts enabled, and at least one available channel of which the generating user is a member.

## NumSidebarCategories

*int64*

The number of sidebar categories to be created.

## NumFollowedThreads

*int64*

The number of threads to follow.

Configuration fields beginning with `Percent` use fractions in the range `[0,1]`, rather than whole-number percentages.

## PercentReplies

*float64*

The percentage of replies (over the total number of posts) to be created.

## PercentRepliesInLongThreads

*float64*

The percentage of replies (over the total number of reply posts) to be created in long running threads.

## PercentUrgentPosts

*float64*

The percentage of posts (over the total number of posts) to be marked with urgent priority.

## PercentRecurringScheduledPosts

*float64*

The probability that each generated scheduled post recurs weekly. The value must be a fraction in the range `[0,1]` and defaults to `0.1`. When `NumScheduledPosts` is non-zero, a non-zero value requires a server version that supports recurring scheduled posts. Recurring posts use the user's valid preferred IANA timezone when available, or one of `UTC`, `America/New_York`, `Europe/London`, and `Asia/Tokyo` as a fallback.

For small scheduled-post targets, the generated recurring proportion may not exactly match this configured probability.

## ChannelMembersDistribution

*[]gencontroller.ChannelMemberDistribution*

The distribution of memberships in channels by maximum number of users in them.

### MemberLimit

The maximum number of users a channel in this group can have. A value of 0 means there is no limit.

### PercentChannels

The percentage of channels that will have the limit configured above.

### Probability

The frequency with which this group of channels will be chosen.

#### Note

The total sum of channels percentages must be equal to 1.
