package miio

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"net"
)

const (
	gatewayPort = 9898
	cmdAck      = "_ack"
)

var (
	iv = []byte{0x17, 0x99, 0x6d, 0x09, 0x3d, 0x28, 0xdd, 0xb3, 0xba, 0x69, 0x5a, 0x2e, 0x6f, 0x58, 0x56, 0x2e}
)

// DeviceUpdateMessage contains data about an update.
type DeviceUpdateMessage struct {
	ID    string
	State interface{}
}

// GatewayState defines the gateway state.
type GatewayState struct {
	On         bool
	RGB        color.Color
	Brightness uint8

	internalRGB uint32
}

// Gateway represents a Xiaomi gateway.
type Gateway struct {
	XiaomiDevice
	multiCast *net.UDPConn
	aesKey    []byte

	State      *GatewayState
	UpdateChan chan *DeviceUpdateMessage

	Sensors  map[string]*SensorHT
	Magnets  map[string]*Magnet
	Motions  map[string]*Motion
	Switches map[string]*Switch
}

// NewGateway creates a new gateway.
func NewGateway(deviceIP, aesKey string) (*Gateway, error) {
	g := &Gateway{
		UpdateChan: make(chan *DeviceUpdateMessage, 50),
		State:      &GatewayState{RGB: color.RGBA{R: 0, G: 0, B: 0, A:0}},
		Sensors:    make(map[string]*SensorHT),
		Magnets:    make(map[string]*Magnet),
		Motions:    make(map[string]*Motion),
		Switches:   make(map[string]*Switch),
		aesKey:     []byte(aesKey),
	}

	err := g.start(deviceIP, "", gatewayPort)
	if err != nil {
		g.stop()
		return nil, err
	}

	err = g.getDevices()
	if err != nil {
		g.stop()
		return nil, err
	}

	err = g.startMultiCast()
	if err != nil {
		g.stop()
		return nil, err
	}

	go g.processMessages()
	go g.processGatewayMessages()
	return g, nil
}

// GetUpdateMessage returns a state update message.
func (g *Gateway) GetUpdateMessage() *DeviceUpdateMessage {
	return &DeviceUpdateMessage{
		State: g.State,
		ID:    g.deviceID,
	}
}

// SetColor sets the LED light color.
func (g *Gateway) SetColor(c color.Color) error {
	r, gC, b, _ := c.RGBA()
	_, _, _, a := g.State.RGB.RGBA()

	corrected := color.RGBA{
		R: uint8(r),
		G: uint8(gC),
		B: uint8(b),
		A: uint8(a),
	}

	data := map[string]interface{}{
		fieldRGB.String(): uint32FromColor(corrected),
	}
	return g.stateCommand(data)
}

// SetBrightness sets the LED light brightness.
func (g *Gateway) SetBrightness(br uint8) error {
	if br > 100 {
		br = 100
	}

	if 0 == br {
		return g.Off()
	}

	r, gC, b, _ := g.State.RGB.RGBA()
	return g.SetColor(color.RGBA{R: uint8(r), G: uint8(gC), B: uint8(b), A: br})
}

// On turns the gateway on.
func (g *Gateway) On() error {
	if !g.State.On {
		return g.SetColor(color.RGBA{255, 255, 255, 0})
	}

	return nil
}

// Off turns the gateway off.
func (g *Gateway) Off() error {
	if g.State.On {
		return g.SetColor(color.RGBA{0, 0, 0, 0})
	}

	return nil
}

// Stop stops the gateway.
func (g *Gateway) Stop() {
	if nil != g.multiCast {
		g.multiCast.Close()
	}
	g.stop()
	close(g.UpdateChan)
}

// UpdateState updates the gateway state.
func (g *Gateway) UpdateState() {
	g.State.internalRGB = g.GetFieldValueUint32(fieldRGB, g.State.internalRGB)

	if g.State.internalRGB > 0 {
		g.State.On = true
		g.State.RGB = rgbFromUint32(g.State.internalRGB)
		_, _, _, a := g.State.RGB.RGBA()
		if a > 100 {
			a = 100
		}

		g.State.Brightness = uint8(100 - a)
	} else {
		g.State.On = false
	}
}

// Requests a list of connected devices.
func (g *Gateway) getDevices() error {
	return g.command(cmdGetDevices, map[string]interface{}{})
}

// Processes incoming messages.
func (g *Gateway) processMessages() {
	for msg := range g.messages {
		m := msg.(*command)
		switch m.Cmd {
		case cmdGetDevices:
			g.processDiscovery(m.Data)
			g.processHandShake(m)
			g.command(cmdGetDeviceState, map[string]interface{}{})
		case cmdGetDeviceState, cmdDeviceReport, cmdSetDeviceState:
			go g.processDeviceState(m)
		case cmdHandShake, cmdHeartBeat:
			go g.processHandShake(m)
		}
	}
}

// Processes handshake response.
func (g *Gateway) processHandShake(cmd *command) {
	if "" != cmd.Token {
		g.token = cmd.Token
	}
}

// Processes a discovery message.
func (g *Gateway) processDiscovery(data string) {
	devIDs := make([]string, 0)
	err := json.Unmarshal([]byte(data), &devIDs)
	if err != nil {
		LOGGER.Error("Failed to get device list: %s", err.Error())
		return
	}

	for _, v := range devIDs {
		g.commandWithSid(cmdGetDeviceState, map[string]interface{}{}, v)
	}
}

// Processes a device state message.
func (g *Gateway) processDeviceState(cmd *command) {
	g.Lock()
	defer g.Unlock()

	mod, err := gatewayDeviceModelString(cmd.Model)
	if err != nil {
		LOGGER.Error("Unknown device type: %s", cmd.Model)
		return
	}

	data := make(map[string]interface{})
	err = json.Unmarshal([]byte(cmd.Data), &data)
	if err != nil {
		LOGGER.Error("Failed to un-marshal device data: %s", err.Error())
		return
	}

	var device IDevice

	switch mod {
	case devGateway:
		device = g
	case devSensorHT:
		d, ok := g.Sensors[cmd.Sid]
		if !ok {
			d = &SensorHT{
				State:   &SensorHTState{},
				Gateway: g,
				ID:      cmd.Sid,
			}
			g.Sensors[cmd.Sid] = d
		}
		device = d
	case devMagnet:
		d, ok := g.Magnets[cmd.Sid]
		if !ok {
			d = &Magnet{
				State:   &MagnetState{},
				Gateway: g,
				ID:      cmd.Sid,
			}
			g.Magnets[cmd.Sid] = d
		}
		device = d
	case devMotion:
		d, ok := g.Motions[cmd.Sid]
		if !ok {
			d = &Motion{
				State:   &MotionState{},
				Gateway: g,
				ID:      cmd.Sid,
			}
			g.Motions[cmd.Sid] = d
		}
		device = d
	case devSwitch:
		d, ok := g.Switches[cmd.Sid]
		if !ok {
			d = &Switch{
				State:   &SwitchState{},
				Gateway: g,
				ID:      cmd.Sid,
			}
			g.Switches[cmd.Sid] = d
		}
		device = d
	default:
		LOGGER.Warn("Unsupported device type: %s", cmd.Model)
		return
	}

	device.SetRawState(data)
	device.UpdateState()
	g.UpdateChan <- device.GetUpdateMessage()
}

// Starts multi-cast listener.
func (g *Gateway) startMultiCast() error {
	l, err := net.ListenMulticastUDP("udp4", nil, &net.UDPAddr{
		IP:   net.IPv4(224, 0, 0, 50),
		Port: gatewayPort,
	})

	if err != nil {
		return err
	}

	g.multiCast = l
	go g.listenMultiCast()
	return nil
}

// Listens for a multi-cast messages.
func (g *Gateway) listenMultiCast() {
	buf := make([]byte, 2048)
	for {
		size, _, err := g.multiCast.ReadFromUDP(buf)
		if err != nil {
			LOGGER.Error("Error reading from MultiCast: %s", err.Error())
			return
		}

		if size > 0 {
			LOGGER.Debug("Received device message: %s", string(buf[0:size]))
			msg := make([]byte, size)
			copy(msg, buf[0:size])
			g.conn.DeviceMessages <- msg
		}
	}
}

// Requests a device state.
func (g *Gateway) stateCommand(data map[string]interface{}) error {
	return g.commandWithSid(cmdSetDeviceState, data, g.deviceID)
}

// Performs a device command.
func (g *Gateway) command(cmd string, data map[string]interface{}) error {
	return g.commandWithSid(cmd, data, g.deviceID)
}

// Performs a device command with specific SID.
func (g *Gateway) commandWithSid(cmd string, data map[string]interface{}, sid string) error {
	if "" != sid && "" == g.token {
		return errors.New("empty token")
	}

	block, err := aes.NewCipher(g.aesKey)
	if err != nil {
		LOGGER.Error("Failed to create CMD cipher: %s", err.Error())
		return err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	cipherText := make([]byte, len(g.token))
	mode.CryptBlocks(cipherText, []byte(g.token))
	command := &command{
		deviceDTO: deviceDTO{
			Sid: sid,
		},
		Cmd: cmd,
	}

	data["key"] = fmt.Sprintf("%X", cipherText)
	bytes, _ := json.Marshal(data)
	command.Data = string(bytes)
	err = g.conn.Send(command)
	if err != nil {
		LOGGER.Error("Failed to send CMD: %s", err.Error())
		return err
	}

	return nil
}

// Processes incoming gateway messages.
func (g *Gateway) processGatewayMessages() {
	for msg := range g.conn.DeviceMessages {
		cmd := &command{}
		err := json.Unmarshal(msg, cmd)
		if err != nil {
			LOGGER.Error("Failed to un-marshal command: %s", err.Error())
			continue
		}

		if len(cmd.Cmd) >= len(cmdAck) && cmd.Cmd[len(cmd.Cmd)-len(cmdAck):] == cmdAck {
			cmd.Cmd = cmd.Cmd[:len(cmd.Cmd)-len(cmdAck)]
		}

		if "" == g.deviceID {
			g.deviceID = cmd.Sid
		}

		g.messages <- cmd
	}
}
