package miio

// SensorHTState describes a state of the humidity-temperature sensor.
type SensorHTState struct {
	Temperature float64
	Humidity    float64
	Battery     float32
}

// SensorHT defines a Xiaomi humidity-temperature sensor.
type SensorHT struct {
	XiaomiDevice

	ID      string
	Gateway *Gateway
	State   *SensorHTState
}

// Stops is not uses for gateway devices.
func (s *SensorHT) Stop() {
}

// GetUpdateMessage returns device's state update message.
func (s *SensorHT) GetUpdateMessage() *DeviceUpdateMessage {
	return &DeviceUpdateMessage{
		ID:    s.ID,
		State: s.State,
	}
}

// UpdateState performs a device update.
func (s *SensorHT) UpdateState() {
	s.State.Temperature = s.GetFieldPercentage(fieldName, s.State.Temperature)
	s.State.Humidity = s.GetFieldPercentage(fieldHumidity, s.State.Humidity)
	s.State.Battery = s.GetBatteryLevel(s.State.Battery)
}
