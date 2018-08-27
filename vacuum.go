package miio

import (
	"encoding/json"
	"time"
)

const (
	// Number of command retries.
	vacRetries = 3
)

// VacError defines possible vacuum error.
type VacError int

const (
	// VacErrorNo describes no errors.
	VacErrorNo VacError = iota
	// VacErrorCharge describes error with charger.
	VacErrorCharge
	// VacErrorFull describes full dust container.
	VacErrorFull
	// VacErrorUnknown describes unknown error
	VacErrorUnknown
)

// VacState defines possible vacuum state.
type VacState int

const (
	// VacStateUnknown describes unknown state.
	VacStateUnknown VacState = iota
	// VacStateInitiating indicates that vacuum is in initializing mode.
	VacStateInitiating
	// VacStateSleeping indicates that vacuum is in a sleep mode.
	VacStateSleeping
	// VacStateWaiting indicates that vacuum is in waiting mode.
	VacStateWaiting
	// VacStateCleaning indicates that vacuums is cleaning.
	VacStateCleaning
	// VacStateReturning indicates that vacuum is returning to the dock.
	VacStateReturning
	// VacStateCharging indicates that vacuum is charging.
	VacStateCharging
	// VacStatePaused indicates that cleaning is paused.
	VacStatePaused
	// VacStateSpot indicates that vacuum is cleaning a spot.
	VacStateSpot
	//VacStateShuttingDown indicates that vacuum is shutting down.
	VacStateShuttingDown
	// VacStateUpdating indicates that vacuum is in an update mode.
	VacStateUpdating
	// VacStateDocking indicates that vacuum is in a process of docking.
	VacStateDocking
	// VacStateZone indicates that vacuum is cleaning az one.
	VacStateZone
	// VacStateFull indicates that dust bag is full.
	VacStateFull
)

// VacuumState describes a vacuum state.
type VacuumState struct {
	Battery    int
	CleanArea  int
	CleanTime  int
	IsDND      bool
	IsCleaning bool
	FanPower   int
	Error      VacError
	State      VacState
}

// Vacuum state obtained from the device.
type internalState struct {
	Battery    int `json:"battery"`
	CleanArea  int `json:"clean_area"`
	CleanTime  int `json:"clean_time"`
	DNDEnabled int `json:"dnd_enabled"`
	ErrorCode  int `json:"error_code"`
	Cleaning   int `json:"cleaning"`
	FanPower   int `json:"fan_power"`
	MapPresent int `json:"map_present"`
	MsgVer     int `json:"msg_ver"`
	MsgSeq     int `json:"msg_seq"`
	State      int `json:"state"`
}

// Response from the vacuum.
type stateResponse struct {
	Result []*internalState `json:"result"`
}

// Vacuum defines a Xiaomi vacuum cleaner.
type Vacuum struct {
	XiaomiDevice
	State *VacuumState

	UpdateChan chan *DeviceUpdateMessage
}

// NewVacuum creates a new vacuum.
func NewVacuum(deviceIP, token string) (*Vacuum, error) {
	v := &Vacuum{
		State: &VacuumState{},
		XiaomiDevice: XiaomiDevice{
			rawState: make(map[string]interface{}),
		},
	}

	err := v.start(deviceIP, token, defaultPort)
	if err != nil {
		return nil, err
	}

	go v.processUpdates()
	v.UpdateChan = make(chan *DeviceUpdateMessage, 100)
	return v, nil
}

// Stop stops the device.
func (v *Vacuum) Stop() {
	v.stop()
	close(v.UpdateChan)
}

// GetUpdateMessage returns an update message.
func (v *Vacuum) GetUpdateMessage() *DeviceUpdateMessage {
	return &DeviceUpdateMessage{
		ID:    v.deviceID,
		State: v.State,
	}
}

// UpdateState performs a state update.
func (v *Vacuum) UpdateState() {
	v.Lock()
	defer v.Unlock()

	b, ok := v.rawState[cmdGetStatus]
	if !ok {
		return
	}

	r := &stateResponse{}
	err := json.Unmarshal(b.([]byte), r)
	if err != nil {
		LOGGER.Error("Failed to un-marshal vacuum response: %s", err.Error())
		return
	}

	if 0 == len(r.Result) {
		return
	}

	v.State.Battery = r.Result[0].Battery
	v.State.CleanArea = r.Result[0].CleanArea
	v.State.CleanTime = r.Result[0].CleanTime
	v.State.IsDND = r.Result[0].DNDEnabled != 0
	v.State.IsCleaning = r.Result[0].Cleaning != 0
	v.State.FanPower = r.Result[0].FanPower

	switch r.Result[0].ErrorCode {
	case 0:
		v.State.Error = VacErrorNo
	case 9:
		v.State.Error = VacErrorCharge
	case 100:
		v.State.Error = VacErrorFull
	default:
		v.State.Error = VacErrorUnknown
	}

	switch r.Result[0].State {
	case 1:
		v.State.State = VacStateInitiating
	case 2:
		v.State.State = VacStateSleeping
	case 3:
		v.State.State = VacStateWaiting
	case 5:
		v.State.State = VacStateCleaning
	case 6:
		v.State.State = VacStateReturning
	case 8:
		v.State.State = VacStateCharging
	case 9:
		v.State.State = VacStatePaused
	case 11:
		v.State.State = VacStateSpot
	case 13:
		v.State.State = VacStateShuttingDown
	case 14:
		v.State.State = VacStateUpdating
	case 15:
		v.State.State = VacStateDocking
	case 17:
		v.State.State = VacStateZone
	case 100:
		v.State.State = VacStateFull
	default:
		v.State.State = VacStateUnknown
	}

	v.UpdateChan <- v.GetUpdateMessage()
}

// UpdateStatus requests for a state update.
func (v *Vacuum) UpdateStatus() bool {
	return v.sendCommand(cmdGetStatus, nil, true, vacRetries)
}

// StartCleaning starts the cleaning cycle.
func (v *Vacuum) StartCleaning() bool {
	if !v.sendCommand(cmdStart, nil, false, vacRetries) {
		return false
	}

	time.Sleep(1 * time.Second)
	return v.UpdateStatus()
}

// PauseCleaning pauses the cleaning cycle.
func (v *Vacuum) PauseCleaning() bool {
	if !v.sendCommand(cmdPause, nil, false, vacRetries) {
		return false
	}

	time.Sleep(1 * time.Second)
	return v.UpdateStatus()
}

// StopCleaning stops the cleaning cycle.
func (v *Vacuum) StopCleaning() bool {
	if !v.sendCommand(cmdStop, nil, false, vacRetries) {
		return false
	}

	time.Sleep(1 * time.Second)
	return v.UpdateStatus()
}

// StopCleaningAndDock stops the cleaning cycle and returns to dock.
func (v *Vacuum) StopCleaningAndDock() bool {
	if !v.sendCommand(cmdStop, nil, false, vacRetries) {
		return false
	}

	time.Sleep(1 * time.Second)
	if !v.sendCommand(cmdDock, nil, false, vacRetries) {
		return false
	}
	return v.UpdateStatus()
}

// FindMe sends the find me command.
func (v *Vacuum) FindMe() bool {
	return v.sendCommand(cmdFindMe, nil, false, vacRetries)
}

// SetFanSpeed sets fan speed
func (v *Vacuum) SetFanPower(val uint8) bool {
	if val > 100 {
		val = 100
	}
	if !v.sendCommand(cmdFanPower, []interface{}{val}, false, vacRetries) {
		return false
	}

	return v.UpdateStatus()
}

// Processes internal updates.
// We care only about state update messages.
func (v *Vacuum) processUpdates() {
	for msg := range v.messages {
		m := msg.(string)
		switch m {
		case cmdGetStatus:
			v.UpdateState()
		}
	}
}
