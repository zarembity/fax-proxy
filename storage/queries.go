package storage

const (
	fetchList = `SELECT 
       id,
       start_stamp,
       uuid,
       file_name,
       caller_id_number,
       destination_number,
       hangup_cause,
       destination,
       result,
       total_pages,
       result_pages,
       bad_row,
       fax_result_text,
       fax_result_code
	FROM fax WHERE %s`

	InsertQuery = `INSERT INTO "fax" 
	("uuid", "file_name", "caller_id_number", "destination_number", "destination", "result", "fax_result_code") 
	VALUES 
	('%s', '%s', '%s', '%s', '%s', '%s', %d)`
)
