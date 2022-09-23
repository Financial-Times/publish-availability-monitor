package httpcaller

import (
	"strings"
)

func ConstructPamTID(tid string) string {
	if strings.HasPrefix(tid, "tid_") {
		tid = tid[:4] + "pam_" + tid[4:]
	}

	return tid
}
