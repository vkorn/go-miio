//go:generate enumer -type=gatewayDeviceModel -transform=snake -trimprefix=dev
//go:generate enumer -type=fldName -transform=snake -trimprefix=field
//go:generate enumer -type=internalClick -transform=snake -trimprefix=cl

package miio

const (
	cmdGetDevices     = "get_id_list"
	cmdGetDeviceState = "read"
	cmdDeviceReport   = "report"
	cmdSetDeviceState = "write"
	cmdHandShake      = "handshake"
	cmdHeartBeat      = "heartbeat"

	cmdGetStatus = "get_status"
	cmdStart     = "app_start"
	cmdStop      = "app_stop"
	cmdPause     = "app_pause"
	cmdDock      = "app_charge"
	cmdFindMe    = "find_me"
)

// Gateway device model.
type gatewayDeviceModel int

const (
	devGateway gatewayDeviceModel = iota
	devSwitch
	devSensorHT
	devMagnet
	devMotion
)

// Field names.
type fldName int

const (
	fieldRGB fldName = iota
	fieldName
	fieldHumidity
	fieldVoltage
	fieldStatus
	fieldNoMotion
)

// ClickType represents Xiaomi switch state.
type ClickType int

const (
	// ClickNo describes no click.
	ClickNo ClickType = iota
	// ClickSingle describes single click.
	ClickSingle
	// ClickDouble describes double click.
	ClickDouble
	// ClickLongPress describes the beginning of a long click.
	ClickLongPress
	// ClickLongRelease describes the end of a long click.
	ClickLongRelease
)

// Internal click type.
type internalClick int

const (
	clClick internalClick = iota
	clDoubleClick
	clLongClickPress
	clLongClickRelease
)
