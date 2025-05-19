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

The number of Custom Profile Attribute fieldsto be created.

## NumPosts

*int64*

The number of posts to be created.

## NumReactions

*int64*

The number of reactions to be created.

## NumPostReminders

*int64*

The number of post reminders to be created.

## NumSidebarCategories

*int64*

The number of sidebar categories to be created.

## NumFollowedThreads

*int64*

The number of threads to follow.

## PercentReplies

*float64*

The percentage of replies (over the total number of posts) to be created.

## PercentRepliesInLongThreads

*float64*

The percentage of replies (over the total number of reply posts) to be created in long running threads.

## PercentUrgentPosts

*float64*

The percentage of posts (over the total number of posts) to be marked with urgent priority.

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
