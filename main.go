package main

import (
	"./arduino"
	"./player"
	"./webserver"
	"github.com/Sirupsen/logrus"
	"github.com/facebookgo/inject"
)

type Startable interface {
	Start() error
}

func main() {

	services := make([]interface{}, 0)

	services = append(
		services,
		&arduino.Arduino{},
		&webserver.Webserver{},
		&player.Player{},
	)
	err := inject.Populate(services...)
	if err != nil {
		panic(err)
	}

	for _, s := range services {
		if s, ok := s.(Startable); ok {
			logrus.Info("Starting")
			s.Start()
		}
	}

	//p.Play(2)
	//<-time.After(time.Second * 3)
	//p.Play(3)

	select {}
}
