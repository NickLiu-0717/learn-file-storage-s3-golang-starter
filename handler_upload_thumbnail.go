package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coundn't parse multi part form", err)
		return
	}
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediatype := header.Header.Get("Content-Type")
	mediatype, _, err = mime.ParseMediaType(mediatype)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse mediatype", err)
		return
	}
	fileextention := strings.Split(mediatype, "/")
	if fileextention[0] != "image" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", errors.New("need image/png or image/jpeg"))
		return
	}

	filename := fmt.Sprintf("%s.%s", videoID, fileextention[1])
	dstfile, err := os.Create(filepath.Join(cfg.assetsRoot, filename))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coundn't create file", err)
		return
	}

	defer dstfile.Close()

	if _, err = io.Copy(dstfile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't copy file", err)
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
	tURL := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, filename)
	video.ThumbnailURL = &tURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coundn't update the video thumbnail", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
