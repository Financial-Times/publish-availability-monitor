package main

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	os.Exit(m.Run())
}

func TestIsMessagePastPublishSLA_pastSLA(t *testing.T) {
	publishDate := time.Now().Add(-(threshold + 1) * time.Second)
	if !isMessagePastPublishSLA(publishDate, threshold) {
		t.Error("Did not detect message past SLA")
	}
}

func TestIsMessagePastPublishSLA_notPastSLA(t *testing.T) {
	publishDate := time.Now()
	if isMessagePastPublishSLA(publishDate, threshold) {
		t.Error("Valid message marked as passed SLA")
	}
}

func TestIsSyntheticMessage_naturalMessage(t *testing.T) {
	if isSyntheticMessage(naturalTID) {
		t.Error("Normal message marked as synthetic")
	}
}

func TestIsSyntheticMessage_syntheticMessage(t *testing.T) {
	if !isSyntheticMessage(syntheticTID) {
		t.Error("Synthetic message marked as normal")
	}
}

const threshold = 120
const syntheticTID = "SYNTHETIC-REQ-MONe4d2885f-1140-400b-9407-921e1c7378cd"
const naturalTID = "tid_xltcnbckvq"