package fs_connector

import (
	"fmt"
	"gitlab.com/a.zaremba/fax-proxy-service/storage"
	"strconv"
	"time"

	"github.com/fiorix/go-eventsocket/eventsocket"
	log "github.com/sirupsen/logrus"
)

func (fs *FsConnector) runReceiveServer() {
	log.Fatal(eventsocket.ListenAndServe(":8082", fs.receive))
}

func (fs *FsConnector) receive(c *eventsocket.Connection) {
	fmt.Println("new client:", c.RemoteAddr())

	_, err := c.Send("connect")
	if err != nil {
		log.Fatalf("receive, command connect return error: %s", err)
	}

	_, err = c.Send("myevents")
	if err != nil {
		log.Fatalf("receive, command myevents return error: %s", err)
	}

	_, err = c.Execute("set", "hangup_after_bridge=true", false)
	if err != nil {
		log.Fatalf("receive, command set hangup_after_bridge=true return error: %s", err)
	}

	_, err = c.Execute("set", "fax_enable_t38_request=true", false)
	if err != nil {
		log.Fatalf("receive, command set fax_enable_t38_request=true return error: %s", err)
	}

	_, err = c.Execute("set", "fax_enable_t38=true", false)
	if err != nil {
		log.Fatalf("receive, command set fax_enable_t38=true return error: %s", err)
	}

	_, err = c.Execute("answer", "", false)
	if err != nil {
		log.Fatalf("receive, command answer return error: %s", err)
	}

	defer c.Close()

	for {
		ev, err := c.ReadEvent()
		if err != nil {
			log.Fatal(err)
		}

		if ev.Get("Event-Name") == "CHANNEL_ANSWER" && ev.Get("Variable_origination_method") == "receive_fax" {
			go fs.answerReceiveHandler(c, ev)
			return
		}
	}

}

func (fs *FsConnector) answerReceiveHandler(rc *eventsocket.Connection, rev *eventsocket.Event) {

	rhUuid := rev.Get("Variable_uuid")

	_, err := rc.Execute("playback", fs.cfg.ReceiveAudio, true)
	if err != nil {
		log.Fatalf("answerReceiveHandler, command playback ReceiveAudio return error: %s", err)
	}
	_, err = rc.Execute("playback", "silence_stream://2000", true)
	if err != nil {
		log.Fatalf("answerReceiveHandler, command playback silence_stream return error: %s", err)
	}

	_, err = rc.Execute("rxfax", fs.cfg.FaxPath+"/"+rhUuid+".tiff", false)
	if err != nil {
		log.Fatalf("answerReceiveHandler, command rxfax return error: %s", err)
	}

	defer rc.Close()
	log.Info("AnswerReceiveHandler UUID: " + rhUuid + " Complete")
}

func (fs *FsConnector) runReceiveResultServer() {
	c, err := eventsocket.Dial(fs.cfg.Addr, fs.cfg.Pass)
	if err != nil {
		log.Fatalf("runReceiveResultServer, eventsocket.Dial return error: %s", err)
	}

	_, err = c.Send("event plain ALL")
	if err != nil {
		log.Fatalf("runReceiveResultServer, event plain ALL return error: %s", err)
	}

	defer c.Close()

	for {
		ev, err := c.ReadEvent()
		if err != nil {
			log.Fatal(err)
		}
		go fs.ReceiveEventProcessor(ev, c)
	}

}

func (fs *FsConnector) ReceiveEventProcessor(event *eventsocket.Event, rc *eventsocket.Connection) {

	//dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
	//	cfg.Connection.Db.DbAddr, cfg.Connection.Db.DbPort, cfg.Connection.Db.DbUser, cfg.Connection.Db.DbPass, cfg.Connection.Db.DbName)
	//db, err := sql.Open("postgres", dbInfo)
	//defer db.Close()
	//
	//INSERTQuery, err := db.Prepare(InsertQuery)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer INSERTQuery.Close()

	if event.Get("Event-Subclass") == "spandsp::rxfaxresult" && event.Get("Variable_origination_method") == "receive_fax" {
		//event.PrettyPrint()
		t := time.Now()
		fmt.Println("\nNew receive fax event")

		UnId := event.Get("Caller-Unique-Id")
		TotalPages := event.Get("Variable_fax_document_total_pages")
		TransferredPages := event.Get("Variable_fax_document_transferred_pages")
		ResultText := event.Get("Variable_fax_result_text")
		ResultCode := event.Get("Variable_fax_result_code")
		FaxSuccess := event.Get("Variable_fax_success")
		BadRows := event.Get("Variable_fax_bad_rows")
		Cause := event.Get("Hangup-Cause")
		FileName := event.Get("Variable_fax_filename")
		Caller := event.Get("Caller-Orig-Caller-Id-Number")
		Destination := event.Get("Caller-Destination-Number")

		result := "in progress"
		if (TotalPages == TransferredPages) && TotalPages != "0" {
			result = "success"
		} else {
			result = "fail"
		}

		resultCode, err := strconv.Atoi(ResultCode)
		if err != nil {
			log.Errorf("parse result code return error: %s", err)
		}

		historyItem := storage.HistoryItem{
			Uuid:           UnId,
			FaxResultCode:  resultCode,
			FileName:       FileName,
			CallerIdNumber: Caller,
			Destination:    Destination,
		}

		err = fs.st.Save(historyItem, "receive", result)
		if err != nil {
			log.Errorf("fs.st.Save return error: %s", err)
		}

		log.Info("spandsp receive UUID: " + UnId + " TotalPages: " + TotalPages + " transfered: " + TransferredPages + " ResultText: " + ResultText + " FaxSuccess: " + FaxSuccess + " BadRows:" + BadRows + " Cause:" + Cause)
	}
}
