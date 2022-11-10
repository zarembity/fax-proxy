package fs_connector

import (
	"database/sql"
	"fmt"
	"github.com/fiorix/go-eventsocket/eventsocket"
	"gitlab.com/a.zaremba/fax-proxy-service/config"
	"log"
)

func RunSendServer() {
	log.Fatal(eventsocket.ListenAndServe(":8083", send))
}

func send(c *eventsocket.Connection) {
	fmt.Println("new send client:", c.RemoteAddr())

	var cfg = &config.Config{}

	err := config.ReadConfig(cfg, *flagConfig)
	if err != nil {
		log.Fatalf("read config return error: %s", err)
	}

	c.Send("connect")  //Всегда должно начинаться с этого
	c.Send("myevents") // подписываюсь на собития этого канала

	for {
		ev, err := c.ReadEvent()
		if err != nil {
			log.Fatal(err)
		}
		if ev.Get("Event-Name") == "CHANNEL_ANSWER" && ev.Get("Variable_origination_method") == "send_fax" {
			go AnswerSendHandler(c, ev)
			return
		}
	}
	c.Close() // закрываю подключение к Event Socket
}

func RunSendResultServer() {

	var cfg = &config.Config{}

	err := config.ReadConfig(cfg, *flagConfig)
	if err != nil {
		log.Fatalf("read config return error: %s", err)
	}

	c, err := eventsocket.Dial(cfg.Connection.FreeSwitch.Addr, cfg.Connection.FreeSwitch.Pass)

	//c.Send("event plain CUSTOM spandsp::txfaxresult")
	c.Send("event plain ALL")

	for {
		ev, err := c.ReadEvent()
		if err != nil {
			log.Fatal(err)
		}
		go SendEventProcessor(ev, c)
	}
	c.Close()
}

func AnswerSendHandler(ac *eventsocket.Connection, aev *eventsocket.Event) {

	var cfg = &config.Config{}

	err := config.ReadConfig(cfg, *flagConfig)
	if err != nil {
		log.Fatalf("read config return error: %s", err)
	}

	EhUuid := aev.Get("Unique-Id")
	//IvrWelcome := aev.Get("Variable_ivrfilewelcome")
	//IvrBye := aev.Get("Variable_ivrfilebye")
	//Dtmf := aev.Get("Variable_dtmf")
	//Sl := aev.Get("Variable_sl")

	ac.Execute("set", "absolute_codec_string=PCMU", false)
	ac.Execute("set", "fax_use_ecm=false", false)
	ac.Execute("set", "fax_verbose=true", false)
	ac.Execute("set", "fax_enable_t38_request=false", false)
	ac.Execute("set", "fax_enable_t38=true", false)

	//if Dtmf != "" && Sl != "" {
	//	log.Info("AnswerSendHandler UUID: " + EhUuid + " dtmf: " + Dtmf + "Sleep: " + Sl)
	//	ac.Execute("sleep", Sl, false)
	//	//ac.Execute("playback", "silence_stream://1000", false)
	//	//ac.Execute("start_dtmf_generate", "", false)
	//	ac.Execute("send_dtmf", Dtmf, false)
	//} else {
	//	if IvrWelcome != "" {
	//		log.Info("AnswerSendHandler UUID: " + EhUuid + " playback " + IvrWelcome)
	//		ac.Execute("playback", cfg.Connection.FreeSwitch.AudioPath+"/"+IvrWelcome, false)
	//	}
	//}
	ac.Execute("txfax", cfg.Connection.FreeSwitch.FaxPath+"/42b88c73-ab81-476c-a69e-d704afee4244.rxfax.tiff", true)
	//if IvrBye != "" {
	//	log.Info("AnswerSendHandler UUID: " + EhUuid + " playback " + IvrBye)
	//	ac.Execute("playback", cfg.Connection.FreeSwitch.AudioPath+"/"+IvrBye, false)
	//}
	//ac.Execute("hangup", "", false)
	//ac.Close()
	log.Info("AnswerSendHandler UUID: " + EhUuid + " Complete")
}

func SendEventProcessor(event *eventsocket.Event, cs *eventsocket.Connection) {

	var cfg = &config.Config{}

	err := config.ReadConfig(cfg, *flagConfig)
	if err != nil {
		log.Fatalf("read config return error: %s", err)
	}

	dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Connection.Db.DbAddr, cfg.Connection.Db.DbPort, cfg.Connection.Db.DbUser, cfg.Connection.Db.DbPass, cfg.Connection.Db.DbName)
	db, err := sql.Open("postgres", dbInfo)
	defer db.Close()

	if event.Get("Event-Subclass") == "spandsp::txfaxresult" && event.Get("Variable_origination_method") == "send_fax" {
		fmt.Println("\nNew send fax event")
		event.PrettyPrint()

		Uuid := event.Get("Caller-Unique-Id")
		Tpage := event.Get("Fax-Document-Total-Pages")
		TRpage := event.Get("Fax-Document-Transferred-Pages")
		ResultText := event.Get("Fax-Result-Text")
		FaxSuccess := event.Get("Fax-Success")
		//BadR := event.Get("fax_bad_rows")
		//HangupC := event.Get("Variable_proto_specific_hangup_cause")
		//cs.Send()
		log.Info("spandsp sender UUID: " + Uuid + " total: " + Tpage + " transfered: " + TRpage + " result_text: " + ResultText + " succes: " + FaxSuccess)
		//cs.Execute("log", "CRIT spandsp: sender UUID: " + Uuid + " total: " + Tpage + " transfered: "+TRpage+" result_text: "+ResultText+" succes: "+FaxSuccess, false)
		//cs.ExecuteUUID( Uuid , "log", "CRIT spandsp: sender UUID: " + Uuid + " total: " + Tpage + " transfered: "+TRpage+" result_text: "+ResultText+" succes: "+FaxSuccess)
		//cs.ExecuteUUID(Uuid ,"playback", cfg.Connection.FreeSwitch.AudioPath+"/Variable_ivrfilebye"+IvrBye)
	} else if event.Get("Variable_origination_method") == "send_fax" && event.Get("Answer-State") == "answered" {
		fmt.Println("\nNew answer send fax event")
		event.PrettyPrint()
	} else if event.Get("Variable_origination_method") == "send_fax" && event.Get("Answer-State") == "hangup" {
		fmt.Println("\nNew hangup send fax event")
		event.PrettyPrint()
	}

}
