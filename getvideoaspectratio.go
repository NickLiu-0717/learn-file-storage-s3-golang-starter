package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var videoInfo map[string]interface{}
	err = json.Unmarshal(out.Bytes(), &videoInfo)
	if err != nil {
		return "", err
	}
	var width float64
	var height float64
	if streams, ok := videoInfo["streams"].([]interface{}); ok {
		for _, stream := range streams {
			if streamMap, ok := stream.(map[string]interface{}); ok {
				if w, ok := streamMap["width"].(float64); ok {
					width = w
				}
				if h, ok := streamMap["height"].(float64); ok {
					height = h
				}

			}
		}
	}

	if width/height < 1.78 && width/height > 1.76 {
		return "16/9", nil
	} else if width/height < 0.568 && width/height > 0.557 {
		return "9/16", nil
	} else {
		return "other", nil
	}
}
