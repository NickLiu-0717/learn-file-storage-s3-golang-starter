package main

import (
	"testing"
)

func TestGetVideoRaio(t *testing.T) {
	filepath1 := "samples/boots-video-horizontal.mp4"
	ratio1 := "16/9"

	gotRatio, err := getVideoAspectRatio(filepath1)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	if gotRatio != ratio1 {
		t.Errorf("Wrong answer: got: %s, need: %s", gotRatio, ratio1)
	}

	filepath2 := "samples/boots-video-vertical.mp4"
	ratio2 := "9/16"

	gotRatio, err = getVideoAspectRatio(filepath2)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	if gotRatio != ratio2 {
		t.Errorf("Wrong answer: got: %s, need: %s", gotRatio, ratio2)
	}
}
