package main

import (
	"flag"
	"gitlab.com/a.zaremba/fax-proxy-service/storage"
	"io"
	"os"

	"gitlab.com/a.zaremba/fax-proxy-service/config"
	fc "gitlab.com/a.zaremba/fax-proxy-service/fs-connector"
	"gitlab.com/a.zaremba/fax-proxy-service/http-server"
	"gitlab.com/a.zaremba/fax-proxy-service/manager"
	sg "gitlab.com/a.zaremba/fax-proxy-service/server-group"

	log "github.com/sirupsen/logrus"
)

var (
	flagConfig     = flag.String("config-file", "config.yaml", "config file name")
	flagUploadPath = flag.String("upload-path", "/tmp", "path to save files")
)

func main() {

	log.Info("Starting fax service")

	var forever chan struct{}

	flag.Parse()

	cfg := &config.Config{}

	err := config.ReadConfig(cfg, *flagConfig)
	if err != nil {
		log.Fatalf("read config return error: %s", err)
	}

	f, err := os.OpenFile(cfg.Service.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Log file %s can't open. Error: %s", cfg.Service.Log, err)
	}

	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	err = initServices(cfg)
	if err != nil {
		log.Fatalf("init services return error: %s", err)
	}

	log.Info("all services init success!")

	<-forever
}

func initServices(cfg *config.Config) error {
	group := sg.NewServerGroup()

	st, err := storage.New(cfg)
	if err != nil {
		return err
	}

	fs, err := fc.New(cfg.Connection.FreeSwitch)
	if err != nil {
		return err
	}
	log.Info("freeSwitch event socket client init success!")

	httpServer, err := http_server.New(*flagUploadPath, cfg.Service, fs)
	if err != nil {
		return err
	}
	log.Info("http server init success!")

	mng := manager.NewManager(cfg, httpServer.GetUploadChan(), fs, st)

	group.Add(httpServer)
	group.Add(fs)
	group.Add(mng)

	if err := group.StartAll(); err != nil {
		return err
	}

	return nil
}
