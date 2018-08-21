package main

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/vkorn/go-miio"
)

var LOGGER = miio.LOGGER

type discovery struct {
}

func (d *discovery) start() {
	ifaces, err := net.Interfaces()
	if err != nil {
		LOGGER.Fatal("Failed to get interfaces: %s", err.Error())
	}

	resolver, err := zeroconf.NewResolver(zeroconf.SelectIPTraffic(zeroconf.IPv4),
		zeroconf.SelectIfaces(ifaces))
	if err != nil {
		LOGGER.Fatal("Failed to initialize resolver: %s", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			LOGGER.Info("================")
			LOGGER.Info("Discovered %s over %s.%s", entry.HostName, entry.Service, entry.Domain)
			parts := strings.Split(entry.HostName, "_miio")
			if len(parts) > 1 {
				parts = strings.Split(parts[1], ".")
				id := strings.Trim(parts[0], ".")
				LOGGER.Info("ID: %s", id)
			}
			for _, v := range entry.Text {
				LOGGER.Info(v)
			}

			for _, v := range entry.AddrIPv4 {
				LOGGER.Info("IP: %s", v.String())
			}

			LOGGER.Info("Port: %d", entry.Port)

			LOGGER.Info("================")
		}
		LOGGER.Info("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()
	err = resolver.Browse(ctx, "_miio._udp", "local.", entries)
	if err != nil {
		LOGGER.Fatal("Failed to browse: %s", err.Error())
	}

	<-ctx.Done()

}

func main() {
	LOGGER.Info("Discovering miio devices")
	d := &discovery{}
	d.start()
}
