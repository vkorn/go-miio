package miio

// MagnetState describes a state of the magnet.
type MagnetState struct {
	Opened  bool
	Battery float32
}

// Magnet defines a Xiaomi magnet.
type Magnet struct {
	XiaomiDevice

	ID      string
	Gateway *Gateway
	State   *MagnetState
}

// Stops is not uses for gateway devices.
func (m *Magnet) Stop() {
}

// GetUpdateMessage returns device's state update message.
func (m *Magnet) GetUpdateMessage() *DeviceUpdateMessage {
	return &DeviceUpdateMessage{
		ID:    m.ID,
		State: m.State,
	}
}

// UpdateState performs a device update.
func (m *Magnet) UpdateState() {
	m.State.Battery = m.GetBatteryLevel(m.State.Battery)
	m.State.Opened = m.GetFieldValueBool(fieldStatus, m.State.Opened)
}
