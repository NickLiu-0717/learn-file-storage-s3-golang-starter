package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	uploadLimit := 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, int64(uploadLimit))
	defer r.Body.Close()

	videoString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coundn't get video from database", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User ID doesn't match", err)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse the video", err)
		return
	}
	defer file.Close()

	mediatype := fileHeader.Header.Get("Content-Type")
	mediatype, _, err = mime.ParseMediaType(mediatype)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse media type", err)
		return
	}
	if mediatype != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Wrong file type, need mp4", err)
		return
	}
	fileExtention := strings.Split(mediatype, "/")

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create temp file", err)
		return
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't copy file to tmp file", err)
		return
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't seeking file", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get the video ratio", err)
		return
	}

	fileKey, err := auth.MakeFileKey(fileExtention[1])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't make file key", err)
		return
	}

	switch aspectRatio {
	case "16/9":
		fileKey = "landscape/" + fileKey
	case "9/16":
		fileKey = "portrait/" + fileKey
	default:
		fileKey = "other/" + fileKey
	}

	outputPath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't process video for fast start", err)
		return
	}

	processedFile, err := os.Open(outputPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't open file", err)
		return
	}
	defer processedFile.Close()

	_, err = cfg.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        processedFile,
		ContentType: &mediatype,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't put video on s3", err)
		return
	}

	videoURL := fmt.Sprintf("https://%s/%s", cfg.s3CfDistribution, fileKey)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update the video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
