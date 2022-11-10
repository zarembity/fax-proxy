package fs_connector

import (
	"context"
	"gitlab.com/a.zaremba/fax-proxy-service/storage"
	"time"

	"gitlab.com/a.zaremba/fax-proxy-service/config"

	"github.com/fiorix/go-eventsocket/eventsocket"
	log "github.com/sirupsen/logrus"
)

type FsConnector struct {
	c         *eventsocket.Connection
	cfg       config.FreeSwitch
	eventChan chan string
	st        *storage.Storage
}

func New(cfg config.FreeSwitch, st *storage.Storage) (*FsConnector, error) {
	return &FsConnector{
		eventChan: make(chan string),
		cfg:       cfg,
		st:        st,
	}, nil
}

func (fs *FsConnector) Serve() error {

	log.Info("Start fs connector ...")

	go fs.runReceiveServer()       //слушает сообщения на прием
	go fs.runReceiveResultServer() //слушает сообщения о результате приема

	go RunSendServer()       //слушает сообщения об отправке
	go RunSendResultServer() //слушает сообщения о результате отправки

	return nil
}

func (fs *FsConnector) GetChan() chan string {
	return fs.eventChan
}

func (fs *FsConnector) Stop(ctx context.Context) error {
	log.Info("Stop fs service ...")
	return nil
}

func (fs *FsConnector) SendMessage(message, uid string) (*eventsocket.Event, error) {
	log.Infof("fs sent message: uid: %s, message: %s", uid, message)
	eventResult, err := fs.c.Send(message)
	if err != nil {
		return nil, err
	}

	//var quitListener = make(chan struct{})
	//go func() {
	//	time.Sleep(time.Second * timeOutWorkListener)
	//	close(quitListener)
	//}()
	//
	//go fs.listenEventMessage(uid, quitListener)

	return eventResult, nil
}

func (fs *FsConnector) GenerateTestEvent(message string, delay int) {
	go func() {
		time.Sleep(time.Second * time.Duration(delay))
		fs.eventChan <- message
	}()
}
