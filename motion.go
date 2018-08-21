package miio

import "time"

// MotionState describes a state of the motion sensor.
type MotionState struct {
	Battery    float32
	HasMotion  bool
	LastMotion time.Time
}

// Motion defines a Xiaomi motion sensor.
type Motion struct {
	XiaomiDevice

	ID      string
	Gateway *Gateway
	State   *MotionState
}

// Stops is not uses for gateway devices.
func (m *Motion) Stop() {
}

// GetUpdateMessage returns device's state update message.
func (m *Motion) GetUpdateMessage() *DeviceUpdateMessage {
	return &DeviceUpdateMessage{
		ID:    m.ID,
		State: m.State,
	}
}

// UpdateState performs a device update.
func (m *Motion) UpdateState() {
	m.State.Battery = m.GetBatteryLevel(m.State.Battery)
	m.State.HasMotion = m.GetFieldValueBool(fieldStatus, m.State.HasMotion)
	noMotion := m.GetFieldValueInt32(fieldNoMotion, 0)
	if noMotion > 0 {
		m.State.HasMotion = false
		lastMotion := time.Now().Add(-1 * time.Duration(noMotion) * time.Second)
		m.State.LastMotion = lastMotion
	}

	if m.State.HasMotion {
		m.State.LastMotion = time.Now()
	}
}
