package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFetch(t *testing.T) {
	testCase := []struct {
		uid, phoneNumber, destination, date, result string
	}{
		{
			"4f6bf008-f414-11e9-b5f9-00155db70f20",
			"",
			"",
			"",
			"uuid = '4f6bf008-f414-11e9-b5f9-00155db70f20'",
		},
		{
			"",
			"78121234567",
			"send",
			"",
			"caller_id_number = '78121234567'",
		},
		{
			"",
			"78121234567",
			"receive",
			"2019-10-20",
			"destination_number = '78121234567' AND date(start_stamp) = '2019-10-20'",
		},
		{
			"",
			"",
			"send",
			"2019-10-20",
			"date(start_stamp) = '2019-10-20'",
		},
	}

	for _, caseItem := range testCase {
		assert.Equal(t, caseItem.result,
			makeWhere(caseItem.uid, caseItem.phoneNumber, caseItem.destination, caseItem.date),
		)
	}

}
