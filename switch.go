package miio

// SwitchState describes a state of the switch.
type SwitchState struct {
	Click   ClickType
	Battery float32
}

// Switch defines a Xiaomi switch.
type Switch struct {
	XiaomiDevice

	ID      string
	Gateway *Gateway
	State   *SwitchState
}

// Stops is not uses for gateway devices.
func (s *Switch) Stop() {
}

// GetUpdateMessage returns device's state update message.
func (s *Switch) GetUpdateMessage() *DeviceUpdateMessage {
	return &DeviceUpdateMessage{
		ID:    s.ID,
		State: s.State,
	}
}

// UpdateState performs a device update.
func (s *Switch) UpdateState() {
	s.State.Battery = s.GetBatteryLevel(s.State.Battery)
	clType, err := internalClickString(s.getFieldValue(fieldStatus))
	if err != nil {
		s.State.Click = ClickNo
	}

	switch clType {
	case clClick:
		s.State.Click = ClickSingle
	case clDoubleClick:
		s.State.Click = ClickDouble
	case clLongClickPress:
		s.State.Click = ClickLongPress
	case clLongClickRelease:
		s.State.Click = ClickLongRelease
	default:
		s.State.Click = ClickNo
	}
}
