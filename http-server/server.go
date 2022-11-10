package http_server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/a.zaremba/fax-proxy-service/config"
	"gitlab.com/a.zaremba/fax-proxy-service/fs-connector"
)

type (
	Api struct {
		conf         config.Service
		fileUpload   string
		userFormChan chan UserFormRequest
		fs           *fs_connector.FsConnector
	}

	testFormViewData struct {
		Link string
	}

	UserFormRequest struct {
		UID        string
		Filename   string
		CallNumber string
		DestNumber string
		TimeAdded  time.Time
	}

	uploadResultResponse struct {
		Result string
		Uid    string
		Info   string
	}
)

const (
	errorBadRequest = "Bad request - 505"
)

func New(filePath string, serviceConf config.Service, fs *fs_connector.FsConnector) (api *Api, err error) {
	if filePath == "" {
		return nil, fmt.Errorf("set upload path. Example --upload-path = some/path ")
	}

	return &Api{
		conf:         serviceConf,
		fileUpload:   filePath,
		userFormChan: make(chan UserFormRequest),
		fs:           fs,
	}, nil
}

func (api *Api) GetUploadChan() chan UserFormRequest {
	return api.userFormChan
}

func (api *Api) Serve() error {

	log.Infof("Starting http server on port %d", api.conf.Port)

	serviceMux := http.NewServeMux()
	serviceMux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Pong"))
		if err != nil {
			log.Printf("/ping return error %s", err)
		}
	})

	serviceMux.HandleFunc("/view", api.viewHandler)
	serviceMux.HandleFunc("/send", api.PostOnly(api.BasicAuth(api.sendHandler)))

	//todo отображать только в dev
	serviceMux.HandleFunc("/form-sender", formSender)
	serviceMux.HandleFunc("/form-view", formViewer)
	serviceMux.HandleFunc("/send-test", api.sendHandler)
	serviceMux.HandleFunc("/generate-event", func(w http.ResponseWriter, r *http.Request) {
		messageKeys, ok := r.URL.Query()["message"]
		if !ok || len(messageKeys[0]) < 1 {
			renderError(w, "Url Param 'message' is missing", http.StatusBadRequest)
			return
		}

		message := messageKeys[0]
		if message == "" {
			renderError(w, "'message' can't be empty", http.StatusBadRequest)
			return
		}

		delay := 1
		delayKeys, ok := r.URL.Query()["delay"]
		if ok && len(delayKeys[0]) > 0 {
			dl, errD := strconv.Atoi(delayKeys[0])
			if errD != nil {
				renderError(w, "'delay' only int", http.StatusBadRequest)
				return
			}
			delay = dl
		}

		api.fs.GenerateTestEvent(message, delay)

		_, err := w.Write([]byte("Generated event success"))
		if err != nil {
			log.Printf("/generated event error %s", err)
		}

	})

	err := http.ListenAndServe(fmt.Sprintf(":%d", api.conf.Port), accessLogMiddleware(serviceMux))
	if err != nil {
		return fmt.Errorf("start listen service return err %s", err)
	}

	return nil
}

func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.WithFields(log.Fields{
			"method":     r.Method,
			"RemoteAddr": r.RemoteAddr,
			"URL":        r.URL.Path,
			"time":       time.Since(start),
		}).Info("access")
	})
}

type handlerType func(w http.ResponseWriter, r *http.Request)

func (api *Api) BasicAuth(pass handlerType) handlerType {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, _ := r.BasicAuth()
		if username != "user" || password != "pass" {
			http.Error(w, "Authorization Failed", http.StatusUnauthorized)
			return
		}
		pass(w, r)
	}
}

func (api *Api) PostOnly(h handlerType) handlerType {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h(w, r)
			return
		}
		http.Error(w, "Please use POST only", http.StatusMethodNotAllowed)
	}
}

func (api *Api) Stop(ctx context.Context) error {
	return nil
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(message))
	if err != nil {
		log.Errorf("sendHandler write return error %s", err)
	}
}
