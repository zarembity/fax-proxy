package manager

import (
	"context"
	"gitlab.com/a.zaremba/fax-proxy-service/config"
	"gitlab.com/a.zaremba/fax-proxy-service/fs-connector"
	"gitlab.com/a.zaremba/fax-proxy-service/http-server"
	"gitlab.com/a.zaremba/fax-proxy-service/storage"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	fileUploaderChan chan http_server.UserFormRequest
	freeSwitch       *fs_connector.FsConnector
	cfg              *config.Config
	st               storage.Storage
}

func NewManager(cfg *config.Config, upFile chan http_server.UserFormRequest, fs *fs_connector.FsConnector, st storage.Storage) *Manager {
	return &Manager{
		fileUploaderChan: upFile,
		freeSwitch:       fs,
		cfg:              cfg,
		st:               st,
	}
}

func (m *Manager) Serve() error {

	log.Info("Starting manager ...")

	go m.listenSenderForm()

	return nil
}

func (m *Manager) listenSenderForm() {
	log.Info("manager: start file uploader listener")
	for data := range m.fileUploaderChan {
		log.Infof("listenSenderForm: %+v", data)

		faxMessage := m.freeSwitch.MakeSendCommand(data.UID, data.CallNumber, data.DestNumber)
		eventResult, err := m.freeSwitch.SendMessage(faxMessage, data.UID)
		if err != nil {
			log.Errorf("manager: fs send message return error: %s, message:%s", err, faxMessage)
			return
		}
		log.Infof("sent command: %s, result: %+v", faxMessage, eventResult)

		//TODO Избавиться от создание модели и вынести в канал
		sModel := storage.HistoryItem{
			Uuid:           data.UID,
			CallerIdNumber: data.CallNumber,
			Destination:    data.DestNumber,
		}

		err = m.st.Save(sModel, "send", "in progress")
		if err != nil {
			log.Errorf("save send return error: %s", err)
		}

	}
}

func (m *Manager) Stop(ctx context.Context) error {
	return nil
}
