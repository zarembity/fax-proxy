package fs_connector

//curl -u CallBackUser:WilFjh1GgT2To -d 'method=send_fax&call_number=78127777777&dest_number=061&originate_timeout=40&ivr_file_welcome=test.wav&ivr_file_bye=test.wav' http://127.0.0.1:8084/api
//curl -u CallBackUser:WilFjh1GgT2To -d 'method=send_fax&call_number=78127777777&dest_number=3053631&originate_timeout=40&ivr_file_welcome=ivr-hold_connect_call.wav&ivr_file_bye=ivr-thank_you_for_calling.wav&dtmf=2' http://127.0.0.1:8084/api

const (
	OriginFaxMethod  = "send_fax"
	dtmf             = "2"
	ivrFileWelcome   = "ivr-hold_connect_call.wav"
	IvrFileBye       = "thank_you_for_calling.wav"
	OriginateTimeout = "40"
	gateway          = "test_gateway"
)

//Генерация команды отправки факса
func (fs *FsConnector) MakeSendCommand(uid, callNumber, destNumber string) string {
	channelVars := "{origination_method=send_fax,origination_uuid=" + uid + ",ivrfilewelcome=" + ivrFileWelcome + ",sl=,dtmf=" + dtmf + ",ivrfilebye=" + IvrFileBye + ",hangup_after_bridge=true,originate_timeout=" + OriginateTimeout + ",origination_caller_id_number=" + callNumber + ",origination_caller_id_name=" + callNumber + "}"

	return "bgapi originate " + channelVars + "sofia/gateway/" + gateway + "/" + destNumber + " &socket('localhost:8083 async full')"
}
