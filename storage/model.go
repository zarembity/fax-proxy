package storage

import (
	"database/sql"
	"fmt"
	"gitlab.com/a.zaremba/fax-proxy-service/config"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Storage struct {
		db *sql.DB
	}

	HistoryItem struct {
		Id                int
		StartStamp        time.Time
		Uuid              string
		FileName          string
		CallerIdNumber    string
		DestinationNumber string
		HangupCause       string
		Destination       string
		Result            string
		TotalPages        string
		ResultPages       string
		BadRow            int
		FaxResultText     string
		FaxResultCode     int
	}
)

func New(cfg *config.Config) (*Storage, error) {
	db, err := getConnect(cfg.Connection.Db)
	if err != nil {
		return nil, err
	}

	return &Storage{
		db: db,
	}, nil
}

func getConnect(cfg config.Db) (*sql.DB, error) {
	dbinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DbAddr,
		cfg.DbPort,
		cfg.DbUser,
		cfg.DbPass,
		cfg.DbName,
	)

	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		return nil, err
	}

	return db, err
}

func (s *Storage) Fetch(uid, phoneNumber, destination, date string) (result []HistoryItem, err error) {
	query := fmt.Sprintf(fetchList, makeWhere(uid, phoneNumber, destination, date))
	log.Infof("execute query: [%s]", query)

	rows, err := s.db.Query(query)
	if err != nil {
		return
	}

	for rows.Next() {
		var item HistoryItem
		err = rows.Scan(
			&item.Id,
			&item.StartStamp,
			&item.Uuid,
			&item.FileName,
			&item.CallerIdNumber,
			&item.DestinationNumber,
			&item.HangupCause,
			&item.Destination,
			&item.Result,
			&item.TotalPages,
			&item.ResultPages,
			&item.BadRow,
			&item.FaxResultText,
			&item.FaxResultCode,
		)

		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return
}

func (s *Storage) Save(item HistoryItem, destination, result string) error {
	var lastInsertId int
	sqlQuery := fmt.Sprintf(InsertQuery,
		item.Uuid,
		item.FileName,
		item.CallerIdNumber,
		item.Destination,
		destination,
		result,
		item.FaxResultCode,
	)
	log.Infof("execute sql send query: %s", sqlQuery)

	return s.db.QueryRow(sqlQuery).Scan(&lastInsertId)
}

func makeWhere(uid, phoneNumber, destination, date string) string {
	if uid != "" {
		return fmt.Sprintf("uuid = '%s'", uid)
	}

	var whereList []string

	//todo дописать валидацию
	//if date != "" {
	//	whereList = append(whereList, fmt.Sprintf("date(start_stamp) = '%s'", date))
	//}

	if phoneNumber != "" && destination != "" {
		if destination == "send" {
			whereList = append(whereList, fmt.Sprintf("caller_id_number = '%s'", phoneNumber))
		} else {
			whereList = append(whereList, fmt.Sprintf("destination_number = '%s'", phoneNumber))
		}
	}

	return strings.Join(whereList[:], " AND ")
}
