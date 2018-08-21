package main

import (
	"image/color"
	"time"

	"github.com/vkorn/go-miio"
)

func main() {
	var LOGGER = miio.LOGGER
	ip := "192.168.0.27"
	key := "09F859F7B23A46BE"

	g, err := miio.NewGateway(ip, key)
	if err != nil {
		LOGGER.Fatal("%s", err.Error())
	}

	time.Sleep(3 * time.Second)
	g.Stop()
	g, err = miio.NewGateway(ip, key)
	if err != nil {
		LOGGER.Fatal("%s", err.Error())
	}

	go func() {
		for msg := range g.UpdateChan {
			LOGGER.Info("ID: %s, State: %+v", msg.ID, msg.State)
		}
	}()

	time.Sleep(1 * time.Second)
	g.SetBrightness(59)
	g.SetColor(color.RGBA{R: 128, G: 100, B: 24, A: 0})

	time.Sleep(10 * time.Second)
}
