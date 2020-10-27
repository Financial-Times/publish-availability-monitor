package httpcaller

import (
	"strings"
)

func ConstructPamTxId(txId string) string {
	if strings.HasPrefix(txId, "tid_") {
		txId = txId[:4] + "pam_" + txId[4:]
	}

	return txId
}
