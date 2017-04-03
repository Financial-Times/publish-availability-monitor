package main

import (
	"os"
	"testing"
)

const syntheticTID = "SYNTHETIC-REQ-MONe4d2885f-1140-400b-9407-921e1c7378cd"
const carouselRepublishTID = "tid_ofcysuifp0_carousel_1488384556"
const carouselGeneratedTID = "tid_ofcysuifp0_carousel_1488384556_gentx"
const naturalTID = "tid_xltcnbckvq"

func TestMain(m *testing.M) {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	os.Exit(m.Run())
}

func TestIsIgnorableMessage_naturalMessage(t *testing.T) {
	if isIgnorableMessage(naturalTID) {
		t.Error("Normal message marked as ignorable")
	}
}

func TestIsIgnorableMessage_syntheticMessage(t *testing.T) {
	if !isIgnorableMessage(syntheticTID) {
		t.Error("Synthetic message marked as normal")
	}
}

func TestIsIgnorableMessage_carouselRepublishMessage(t *testing.T) {
	if !isIgnorableMessage(carouselRepublishTID) {
		t.Error("Carousel republish message marked as normal")
	}
}

func TestIsIgnorableMessage_carouselGeneratedMessage(t *testing.T) {
	if !isIgnorableMessage(carouselGeneratedTID) {
		t.Error("Carousel generated message marked as normal")
	}
}
