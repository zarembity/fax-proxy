package http_server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"gitlab.com/a.zaremba/fax-proxy-service/storage"

	"github.com/h2non/filetype"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

func (api *Api) sendHandler(w http.ResponseWriter, r *http.Request) {
	uid := uuid.NewV1().String()

	makeResponse := func(uid, result, info string) string {
		bytes, err := json.Marshal(uploadResultResponse{result, uid, info})
		if err != nil {
			log.Errorf("sendHandler, json marshal return error %s", err)
			return errorBadRequest
		}

		return string(bytes)
	}

	callNumber := r.FormValue("call-number")
	if !validatePhoneNumber(callNumber) {
		renderError(w, makeResponse("", "error", errorBadRequest), http.StatusBadRequest)
		return
	}

	destNumber := r.FormValue("dest-number")
	if !validatePhoneNumber(destNumber) {
		renderError(w, makeResponse("", "error", errorBadRequest), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("uploadFile")
	if err != nil {
		renderError(w, makeResponse("", "bad_request", "INVALID_FILE"), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if handler.Size > (api.conf.Image.MaxSize * 1024 * 1024) {
		message := makeResponse("", "bad_request", fmt.Sprintf("Max file size - %v", api.conf.Image.MaxSize))
		renderError(w, message, http.StatusBadRequest)
		return
	}

	filename := generateFilename(api.fileUpload, uid, handler.Filename)
	err = api.saveFile(handler, file, filename)
	if err != nil {
		log.Errorf("http server: Error save file %s", err)
		renderError(w, makeResponse("", "error", "Error save file"), http.StatusBadRequest)
		return
	}

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		deleteFile(filename)
		renderError(w, makeResponse("", "error", "Error save file"), http.StatusBadRequest)
		log.Errorf("http server: read file return error: %s", err)
		return
	}

	kind, err := filetype.Match(buf)
	if err != nil {
		go deleteFile(filename)
		log.Errorf("http server: filetype.Match return error: %s", err)
		renderError(w, makeResponse("", "bad_request", "filetype.Match"), http.StatusBadRequest)
		return
	}

	if !ContainsItemSlice(api.conf.Image.MimeTypes, kind.MIME.Value) {
		go deleteFile(filename)
		rInfo := fmt.Sprintf("You can save onlye %v", api.conf.Image.MimeTypes)
		renderError(w,
			makeResponse("", "error", rInfo),
			http.StatusBadRequest)
		return
	}

	formResult := UserFormRequest{
		UID:        uid,
		Filename:   filename,
		CallNumber: callNumber,
		DestNumber: destNumber,
		TimeAdded:  time.Now(),
	}

	log.Infof("formResult: %+v \n", formResult)

	go func() {
		api.userFormChan <- formResult
	}()

	resp := makeResponse(uid, "success", "")
	log.Info(resp)

	_, err = w.Write([]byte(resp))
	if err != nil {
		log.Errorf("sendHandler write return error %s", err)
	}
}

func formSender(w http.ResponseWriter, r *http.Request) {
	data := testFormViewData{
		Link: "/send-test",
	}

	tmpl, err := template.ParseFiles("tmpl/form-send.html")
	if err != nil {
		log.Errorf("template.ParseFiles test-multipart-form.html return error: %s", err)
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Errorf("http server /test-form return error: %s", err)
	}
}

func formViewer(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("tmpl/form-view.html")
	if err != nil {
		log.Errorf("template.ParseFiles test-multipart-form.html return error: %s", err)
	}

	data := testFormViewData{
		Link: "/view",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Errorf("http server /form-view return error: %s", err)
	}
}

type dataT struct {
	List []storage.HistoryItem
}

func (api Api) viewHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	phoneNumber := r.URL.Query().Get("phone-number")
	destination := r.URL.Query().Get("destination")
	date := r.URL.Query().Get("date")

	rFormat := r.URL.Query().Get("r-format")
	var result []storage.HistoryItem

	showResult := func(items []storage.HistoryItem) {
		var (
			err    error
			result []byte
		)

		if rFormat == "html" {

			tmpl, err := template.ParseFiles("tmpl/view.html")
			if err != nil {
				log.Errorf("template.ParseFiles test-multipart-form.html return error: %s", err)
			}

			data := dataT{List: items}

			err = tmpl.Execute(w, data)
			if err != nil {
				log.Errorf("http server /test-form return error: %s", err)
			}

		} else {
			result, err = json.Marshal(items)
			if err != nil {
				renderError(w, "Server error", http.StatusInternalServerError)
			}
		}

		_, err = w.Write(result)
		if err != nil {
			log.Printf("/generated event error %s", err)
		}
	}

	result, err := storage.Fetch(uid, phoneNumber, destination, date)
	if err == sql.ErrNoRows {
		renderError(w, "Ничего не найдено", http.StatusOK)
		return
	}

	if err != nil {
		log.Infof("fetchOne return error: %s", err)
		renderError(w, "sql query error", http.StatusInternalServerError)
		return
	}

	showResult(result)

}

func (api *Api) saveFile(headers *multipart.FileHeader, file multipart.File, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, file)

	return err
}
