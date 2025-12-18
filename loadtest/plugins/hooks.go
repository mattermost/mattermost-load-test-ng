package plugins

// HookType specifies the type of the hook
type HookType string

// The list of available hook types
const (
	// HookLogin is the [HookType] corresponding to the login hook.
	// It has no corresponding payload struct.
	HookLogin HookType = "hookLogin"

	// HookSwitchTeam is the [HookType] corresponding to the switch team hook.
	// Its payload struct type is [HookPayloadSwitchTeam].
	HookSwitchTeam HookType = "hookSwitchTeam"

	// HookSwitchChannel is the [HookType] corresponding to the switch channel hook.
	// Its payload struct type is [HookPayloadSwitchChannel].
	HookSwitchChannel HookType = "hookSwitchChannel"
)

// HookPayloadSwitchTeam contains the data needed by the switch team hook.
type HookPayloadSwitchTeam struct {
	// ID of the team the user has switched to.
	TeamId string
}

// HookPayloadSwitchTeam contains the data needed by the switch channel hook.
type HookPayloadSwitchChannel struct {
	// ID of the channel the user has switched to.
	ChannelId string
}
