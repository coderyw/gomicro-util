package prefixenum

type PrefixEnum string

const (
	PrefixEnumSchedule PrefixEnum = "ca-schedule-go"

	// color
	PrefixEnumGameColor PrefixEnum = "ca-game-color"

	PrefixEnumGame       PrefixEnum = "ca-game"
	PrefixEnumCaPay      PrefixEnum = "ca-pay"
	PrefixEnumCaBalance  PrefixEnum = "ca-balance"
	PrefixEnumCaActivity PrefixEnum = "ca-activity"
)
