package main

import (
	"fmt"
	"io"
	"net/http"
  "time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	_ "github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

// 10 MB
const maxMemory int = 10 << 20;

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

  err = r.ParseMultipartForm(int64(maxMemory))
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Something went wrong parsing form", err)
    return
  }
  // the above is needed first
  imageData, header, err := r.FormFile("thumbnail");
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Something went wrong parsing form", err)
    return
  }
  defer imageData.Close()

  // generate thumb 
  imageDataInBytes, err := io.ReadAll(imageData);
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Something went wrong parsing image data", err)
    return
  }
  // example "image/png"
  mediaType := header.Header.Get("Content-Type");
  if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
    return
  }
  thumb := thumbnail{
    data: imageDataInBytes, 
    mediaType: mediaType,
  }

  videoMetadata, err := cfg.db.GetVideo(videoID);
  if err != nil {
    respondWithError(w, http.StatusUnauthorized, "Couldn't find video", err);
  }
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

  // add to videomap in memory 
  videoThumbnails[videoID] = thumb;

  // update video in database 
  thumbnailURL := fmt.Sprintf("http://localhost:%d/api/thumbnails/%s", cfg.port, videoID)

  videoMetadata.ThumbnailURL = &thumbnailURL;
  videoMetadata.UpdatedAt = time.Now();

  err = cfg.db.UpdateVideo(videoMetadata);
  if err != nil {
    delete(videoThumbnails, videoID)
    respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
    return
  }

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
