package miio

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nickw444/miio-go/protocol/packet"
)

const (
	// Default device port.
	defaultPort = 54321
)

// IDevice defines Xiaomi device.
type IDevice interface {
	Stop()
	GetUpdateMessage() *DeviceUpdateMessage
	SetRawState(map[string]interface{})
	UpdateState()
}

// XiaomiDevice represents Xiaomi device.
type XiaomiDevice struct {
	sync.Mutex

	conn   *connection
	crypto packet.Crypto

	token    string
	tokenB   []byte
	deviceID string
	rawState map[string]interface{}
	messages chan interface{}

	lastDiscovery time.Time
}

// Sets raw state of the device. Used for Gateway devices.
func (d *XiaomiDevice) SetRawState(state map[string]interface{}) {
	d.rawState = state
}

// Starts listeners.
func (d *XiaomiDevice) start(deviceIP, token string, port int) error {
	c, err := newConnection(deviceIP, port)
	if err != nil {
		return err
	}

	d.messages = make(chan interface{}, 100)
	d.conn = c
	if "" != token {
		d.token = token
		t, err := hex.DecodeString(d.token)
		if err != nil {
			return err
		}

		d.tokenB = t
	}

	return nil
}

// Stops listeners.
func (d *XiaomiDevice) stop() {
	if nil != d.conn {
		close(d.messages)
		d.conn.Close()
	}
}

// Retrieves field value from a response.
func (d *XiaomiDevice) getFieldValue(field fldName) string {
	v, ok := d.rawState[field.String()]
	if !ok {
		return ""
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int:
		return fmt.Sprintf("%d", v.(int))
	case reflect.Float64:
		return fmt.Sprintf("%0.f", v.(float64))
	case reflect.String:
		return v.(string)
	default:
		LOGGER.Warn("Unknown %s value type %s", field, reflect.TypeOf(v).Kind().String())
		return ""
	}
}

// GetFieldValueInt32 returns int32 value.
func (d *XiaomiDevice) GetFieldValueInt32(field fldName, curVal int32) int32 {
	v := d.getFieldValue(field)
	if "" == v {
		return curVal
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		LOGGER.Warn("Failed to parse int: %s", v)
		return curVal
	}
	return int32(n)
}

// GetFieldValueUint32 returns uint32 value.
func (d *XiaomiDevice) GetFieldValueUint32(field fldName, curVal uint32) uint32 {
	v := d.getFieldValue(field)
	if "" == v {
		return curVal
	}

	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		LOGGER.Warn("Failed to parse uint32: %s", v)
		return curVal
	}
	return uint32(n)
}

// GetFieldValueFloat64 returns float64 value.
func (d *XiaomiDevice) GetFieldValueFloat64(field fldName, curVal float64) float64 {
	v := d.getFieldValue(field)
	if "" == v {
		return curVal
	}

	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		LOGGER.Warn("Failed to parse float64: %s", v)
		return curVal
	}
	return n
}

// GetBatteryLevel returns current battery level percent.
func (d *XiaomiDevice) GetBatteryLevel(curVal float32) float32 {
	_, ok := d.rawState[fieldVoltage.String()]
	if !ok {
		return curVal
	}

	return float32(d.GetFieldValueUint32(fieldVoltage, 0)) / 33.0
}

// GetFieldPercentage returns percent field.
func (d *XiaomiDevice) GetFieldPercentage(field fldName, curVal float64) float64 {
	_, ok := d.rawState[field.String()]
	if !ok {
		return curVal
	}

	return d.GetFieldValueFloat64(field, curVal) / 100.0
}

// GetFieldValueBool returns bool value.
func (d *XiaomiDevice) GetFieldValueBool(field fldName, curVal bool) bool {
	v := strings.ToLower(d.getFieldValue(field))
	if "" == v {
		return curVal
	}

	if v == "1" || v == "open" || v == "on" || v == "true" || v == "motion" {
		return true
	}
	return false
}

// Sends the command to a device. Will try to retry.
func (d *XiaomiDevice) sendCommand(cmd string, data map[string]interface{}, storeResponse bool, retries int) bool {
	resp := false
	for ii := 0; ii < retries; ii++ {
		resp = d.doCommand(cmd, data, storeResponse)
		if resp {
			break
		}
	}

	return resp
}

// Performs single command execution.
func (d *XiaomiDevice) doCommand(cmd string, data map[string]interface{}, storeResponse bool) bool {
	if d.lastDiscovery.Add(1 * time.Minute).Before(time.Now()) {
		if false == d.discovery() {
			return false
		}

		d.lastDiscovery = time.Now()
	}

	msgID := time.Now().UTC().Unix()

	c := &deviceCommand{
		ID:     msgID,
		Method: cmd,
		Params: data,
	}
	b, err := json.Marshal(c)
	if err != nil {
		LOGGER.Error("Failed to marshal %s command: %s", cmd, err.Error())
		return false
	}

	p, err := d.crypto.NewPacket(b)
	if err != nil {
		LOGGER.Error("Failed to encrypt %s command: %s", cmd, err.Error())
		return false
	}

	return d.sendAndWait(p, cmd, storeResponse)
}

// Handles discovery request-response.
func (d *XiaomiDevice) discovery() bool {
	d.conn.outMessages <- packet.NewHello().Serialize()
	for {
		select {
		case b := <-d.conn.DeviceMessages:
			if 32 != len(b) {
				LOGGER.Error("Received incorrect discovery package")
				return false
			}

			p, err := packet.Decode(b, nil)
			if err != nil {
				LOGGER.Error("Failed to decode packet: %s", err.Error())
				return false
			}

			if nil == d.crypto {
				c, err := packet.NewCrypto(p.Header.DeviceID, d.tokenB,
					p.Header.Stamp, time.Now().UTC(), clock.New())
				if err != nil {
					LOGGER.Error("Failed to create crypto: %s", err.Error())
					return false
				}

				d.crypto = c
			}

			return true
		case <-time.After(5 * time.Second):
			LOGGER.Error("Timeout while waiting on handshake")
			return false
		}
	}

	return false
}

// Sends a command and waits for a response.
func (d *XiaomiDevice) sendAndWait(p *packet.Packet, cmd string, storeResponse bool) bool {
	d.conn.outMessages <- p.Serialize()
	for {
		select {
		case b := <-d.conn.DeviceMessages:
			p, err := packet.Decode(b, nil)
			if err != nil {
				LOGGER.Error("Failed to decode packet: %s", err.Error())
				return false
			}

			err = p.Verify(d.tokenB)
			if err != nil {
				LOGGER.Error("Failed to verify packet: %s", err.Error())
				continue
			}

			dec, err := d.crypto.Decrypt(p.Data)
			if err != nil {
				LOGGER.Error("Failed to decrypt packet: %s", err.Error())
				continue
			}

			// Trailing \x00
			dec = dec[:len(dec)-1]
			c := &devResponse{}
			err = json.Unmarshal(dec, c)
			if err != nil {
				LOGGER.Error("Failed to un-marshal response: %s", err.Error())
				continue
			}

			if storeResponse {
				d.Lock()
				d.rawState[cmd] = dec
				d.Unlock()
			}

			return true
		case <-time.After(5 * time.Second):
			LOGGER.Error("Timeout while waiting on response fot %s", cmd)
			return false
		}
	}

	return false
}
