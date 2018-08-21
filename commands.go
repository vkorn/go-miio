package miio

// Gateway device definition.
type deviceDTO struct {
	Sid   string `json:"sid,omitempty"`
	Model string `json:"model,omitempty"`
	Data  string `json:"data,omitempty"`
	Token string `json:"token,omitempty"`
}

// Gateway command.
type command struct {
	deviceDTO
	Cmd string `json:"cmd"`
}

// Independent device command.
type deviceCommand struct {
	ID     int64                  `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// Base response from the device.
type devResponse struct {
	ID int64 `json:"id"`
}
