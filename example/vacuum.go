package main

import (
	"time"

	"github.com/vkorn/go-miio"
)

func main() {
	var LOGGER = miio.LOGGER
	ip := "192.168.0.31"
	token := "476d424348304f414776726d3330786e"

	v, err := miio.NewVacuum(ip, token)
	if err != nil {
		LOGGER.Fatal("%s", err.Error())
	}
	v.UpdateStatus()

	time.Sleep(2 * time.Second)
	v.Stop()

	v, err = miio.NewVacuum(ip, token)
	if err != nil {
		LOGGER.Fatal("%s", err.Error())
	}

	go func() {
		for msg := range v.UpdateChan {
			LOGGER.Info("%+v", msg.State)
		}
	}()

	v.UpdateStatus()
	time.Sleep(2 * time.Second)
	v.UpdateStatus()
	time.Sleep(2 * time.Second)
	v.StartCleaning()
	time.Sleep(5 * time.Second)
	v.SetFanPower(80)
	time.Sleep(5 * time.Second)
	v.SetFanPower(40)
	time.Sleep(5 * time.Second)
	v.PauseCleaning()
	time.Sleep(5 * time.Second)
	v.StopCleaningAndDock()
	time.Sleep(3 * time.Second)
	v.FindMe()
}
