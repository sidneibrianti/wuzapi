package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/vincent-petithory/dataurl"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func Find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// Update entry in User map
func updateUserInfo(values interface{}, field string, value string) interface{} {
	log.Debug().Str("field", field).Str("value", value).Msg("User info updated")
	values.(Values).m[field] = value
	return values
}

// webhook for regular messages
func callHook(myurl string, payload map[string]string, id string) {
	log.Info().Str("url", myurl).Msg("Sending POST to client " + id)

	// Log the payload map
	log.Debug().Msg("Payload:")
	for key, value := range payload {
		log.Debug().Str(key, value).Msg("")
	}

	client := clientManager.GetHTTPClient(id)

	format := os.Getenv("WEBHOOK_FORMAT")
	if format == "json" {
		// Send as pure JSON
		// The original payload is a map[string]string, but we want to send the postmap (map[string]interface{})
		// So we try to decode the jsonData field if it exists, otherwise we send the original payload
		var body interface{} = payload
		if jsonStr, ok := payload["jsonData"]; ok {
			var postmap map[string]interface{}
			err := json.Unmarshal([]byte(jsonStr), &postmap)
			if err == nil {
				postmap["token"] = payload["token"]
				body = postmap
			}
		}
		_, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			Post(myurl)
		if err != nil {
			log.Debug().Str("error", err.Error())
		}
	} else {
		// Default: send as form-urlencoded
		_, err := client.R().SetFormData(payload).Post(myurl)
		if err != nil {
			log.Debug().Str("error", err.Error())
		}
	}
}

// webhook for messages with file attachments
func callHookFile(myurl string, payload map[string]string, id string, file string) error {
	log.Info().Str("file", file).Str("url", myurl).Msg("Sending POST")

	client := clientManager.GetHTTPClient(id)

	// Create final payload map
	finalPayload := make(map[string]string)
	for k, v := range payload {
		finalPayload[k] = v
	}

	finalPayload["file"] = file

	log.Debug().Interface("finalPayload", finalPayload).Msg("Final payload to be sent")

	resp, err := client.R().
		SetFiles(map[string]string{
			"file": file,
		}).
		SetFormData(finalPayload).
		Post(myurl)

	if err != nil {
		log.Error().Err(err).Str("url", myurl).Msg("Failed to send POST request")
		return fmt.Errorf("failed to send POST request: %w", err)
	}

	log.Debug().Interface("payload", finalPayload).Msg("Payload sent to webhook")
	log.Info().Int("status", resp.StatusCode()).Str("body", string(resp.Body())).Msg("POST request completed")

	return nil
}

func (s *server) respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// ProcessOutgoingMedia handles media processing for outgoing messages with S3 support
func ProcessOutgoingMedia(userID string, contactJID string, messageID string, data []byte, mimeType string, fileName string, db *sqlx.DB) (map[string]interface{}, error) {
	// Check if S3 is enabled for this user
	var s3Config struct {
		Enabled       bool   `db:"s3_enabled"`
		MediaDelivery string `db:"media_delivery"`
	}
	err := db.Get(&s3Config, "SELECT s3_enabled, media_delivery FROM users WHERE id = $1", userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get S3 config")
		s3Config.Enabled = false
		s3Config.MediaDelivery = "base64"
	}

	// Process S3 upload if enabled
	if s3Config.Enabled && (s3Config.MediaDelivery == "s3" || s3Config.MediaDelivery == "both") {
		// Process S3 upload (outgoing messages are always in outbox)
		s3Data, err := GetS3Manager().ProcessMediaForS3(
			context.Background(),
			userID,
			contactJID,
			messageID,
			data,
			mimeType,
			fileName,
			false, // isIncoming = false for sent messages
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload media to S3")
			// Continue even if S3 upload fails
		} else {
			return s3Data, nil
		}
	}

	return nil, nil
}

// STATUS HELPER FUNCTIONS

// getWAClient gets the WhatsApp client for the current user
func (s *server) getWAClient(r *http.Request) (*MyClient, error) {
	txtid := r.Context().Value("userinfo").(Values).Get("Id")

	if clientManager.GetWhatsmeowClient(txtid) == nil {
		return nil, fmt.Errorf("no active WhatsApp session")
	}

	if !clientManager.GetWhatsmeowClient(txtid).IsConnected() {
		return nil, fmt.Errorf("WhatsApp client not connected")
	}

	// Get the MyClient from clientManager
	mycli := clientManager.GetMyClient(txtid)
	if mycli == nil {
		return nil, fmt.Errorf("client not found")
	}

	return mycli, nil
}

// sendFormattedTextStatus sends a text status with formatting
func (s *server) sendFormattedTextStatus(mycli *MyClient, text string, bgColor, textColor *int64, font *int32) (*types.MessageInfo, error) {
	ctx := context.Background()

	// Create extended text message with formatting
	extendedMsg := &waE2E.ExtendedTextMessage{
		Text: proto.String(text),
	}

	// Add colors if provided (font support will be added later)
	if bgColor != nil {
		extendedMsg.BackgroundArgb = proto.Uint32(uint32(*bgColor))
	}
	if textColor != nil {
		extendedMsg.TextArgb = proto.Uint32(uint32(*textColor))
	}
	// Font support temporarily disabled due to proto incompatibility

	resp, err := mycli.WAClient.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		ExtendedTextMessage: extendedMsg,
	})
	if err != nil {
		return nil, err
	}

	return &types.MessageInfo{
		ID:        resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// sendImageStatus sends an image status
func (s *server) sendImageStatus(mycli *MyClient, imageData []byte, mimeType, caption string) (*types.MessageInfo, error) {
	ctx := context.Background()

	// Upload the image
	uploaded, err := mycli.WAClient.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	// Create image message
	imageMsg := &waE2E.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
	}

	if caption != "" {
		imageMsg.Caption = proto.String(caption)
	}

	resp, err := mycli.WAClient.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		ImageMessage: imageMsg,
	})
	if err != nil {
		return nil, err
	}

	return &types.MessageInfo{
		ID:        resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// sendVideoStatus sends a video status
func (s *server) sendVideoStatus(mycli *MyClient, videoData []byte, mimeType, caption string) (*types.MessageInfo, error) {
	ctx := context.Background()

	// Upload the video
	uploaded, err := mycli.WAClient.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	// Create video message
	videoMsg := &waE2E.VideoMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
	}

	if caption != "" {
		videoMsg.Caption = proto.String(caption)
	}

	resp, err := mycli.WAClient.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		VideoMessage: videoMsg,
	})
	if err != nil {
		return nil, err
	}

	return &types.MessageInfo{
		ID:        resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// sendAudioStatus sends an audio status
func (s *server) sendAudioStatus(mycli *MyClient, audioData []byte, mimeType string, isPTT bool) (*types.MessageInfo, error) {
	ctx := context.Background()

	// Upload the audio
	uploaded, err := mycli.WAClient.Upload(ctx, audioData, whatsmeow.MediaAudio)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	// Create audio message
	audioMsg := &waE2E.AudioMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
		PTT:           proto.Bool(isPTT),
	}

	// Set duration for PTT audio
	if isPTT {
		duration := s.getAudioDuration(audioData, mimeType)
		if duration > 0 {
			audioMsg.Seconds = proto.Uint32(duration)
		}
	}

	resp, err := mycli.WAClient.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		AudioMessage: audioMsg,
	})
	if err != nil {
		return nil, err
	}

	return &types.MessageInfo{
		ID:        resp.ID,
		Timestamp: resp.Timestamp,
	}, nil
}

// syncPrivacySettings syncs privacy settings
func (s *server) syncPrivacySettings(mycli *MyClient) error {
	// Privacy settings sync not implemented yet
	// WhatsApp privacy settings are controlled through the official app
	return nil
}

// Validation functions
func (s *server) isValidFont(font int32) bool {
	validFonts := []int32{0, 1, 2, 6, 7, 8, 9, 10}
	for _, validFont := range validFonts {
		if font == validFont {
			return true
		}
	}
	return false
}

func (s *server) isValidImageMimeType(mimeType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, validType := range validTypes {
		if mimeType == validType {
			return true
		}
	}
	return false
}

func (s *server) isValidVideoMimeType(mimeType string) bool {
	validTypes := []string{
		"video/mp4",
		"video/3gpp",
	}

	for _, validType := range validTypes {
		if mimeType == validType {
			return true
		}
	}
	return false
}

func (s *server) isValidAudioMimeType(mimeType string) bool {
	validTypes := []string{
		"audio/ogg",
		"audio/mpeg",
		"audio/mp4",
		"audio/m4a",
		"audio/aac",
	}

	for _, validType := range validTypes {
		if mimeType == validType {
			return true
		}
	}
	return false
}

// getAudioDuration estimates audio duration
func (s *server) getAudioDuration(audioData []byte, mimeType string) uint32 {
	// Basic estimation: ~1 second per 16KB for compressed audio
	if len(audioData) > 0 {
		return uint32(len(audioData) / 16384)
	}
	return 0
}

// Media processing functions

// processImageSource processes image from different sources
func (s *server) processImageSource(image, source string) ([]byte, string, error) {
	switch source {
	case "base64", "":
		return s.decodeBase64Image(image)
	case "url":
		return s.downloadImageFromURL(image)
	case "file":
		return s.readImageFromFile(image)
	default:
		return nil, "", fmt.Errorf("unsupported source type: %s", source)
	}
}

// processVideoSource processes video from different sources
func (s *server) processVideoSource(video, source string) ([]byte, string, error) {
	switch source {
	case "base64", "":
		return s.decodeBase64Video(video)
	case "url":
		return s.downloadVideoFromURL(video)
	case "file":
		return s.readVideoFromFile(video)
	default:
		return nil, "", fmt.Errorf("unsupported source type: %s", source)
	}
}

// processAudioSource processes audio from different sources
func (s *server) processAudioSource(audio, source string) ([]byte, string, error) {
	switch source {
	case "base64", "":
		return s.decodeBase64Audio(audio)
	case "url":
		return s.downloadAudioFromURL(audio)
	case "file":
		return s.readAudioFromFile(audio)
	default:
		return nil, "", fmt.Errorf("unsupported source type: %s", source)
	}
}

// Base64 decoding functions
func (s *server) decodeBase64Image(data string) ([]byte, string, error) {
	if strings.HasPrefix(data, "data:image") {
		// Handle data URL format
		dataURL, err := dataurl.DecodeString(data)
		if err != nil {
			return nil, "", fmt.Errorf("could not decode base64 data URL: %v", err)
		}
		return dataURL.Data, dataURL.MediaType.ContentType(), nil
	}

	// Handle plain base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", fmt.Errorf("could not decode base64: %v", err)
	}

	// Detect MIME type from content
	mimeType := http.DetectContentType(decoded)
	return decoded, mimeType, nil
}

func (s *server) decodeBase64Video(data string) ([]byte, string, error) {
	if strings.HasPrefix(data, "data:video") {
		dataURL, err := dataurl.DecodeString(data)
		if err != nil {
			return nil, "", fmt.Errorf("could not decode base64 data URL: %v", err)
		}
		return dataURL.Data, dataURL.MediaType.ContentType(), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", fmt.Errorf("could not decode base64: %v", err)
	}

	mimeType := http.DetectContentType(decoded)
	return decoded, mimeType, nil
}

func (s *server) decodeBase64Audio(data string) ([]byte, string, error) {
	if strings.HasPrefix(data, "data:audio") {
		dataURL, err := dataurl.DecodeString(data)
		if err != nil {
			return nil, "", fmt.Errorf("could not decode base64 data URL: %v", err)
		}
		return dataURL.Data, dataURL.MediaType.ContentType(), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", fmt.Errorf("could not decode base64: %v", err)
	}

	mimeType := http.DetectContentType(decoded)
	return decoded, mimeType, nil
}

// URL downloading functions
func (s *server) downloadImageFromURL(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func (s *server) downloadVideoFromURL(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download video: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read video data: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func (s *server) downloadAudioFromURL(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download audio: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download audio: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read audio data: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

// File reading functions
func (s *server) readImageFromFile(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image file: %v", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}

func (s *server) readVideoFromFile(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read video file: %v", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}

func (s *server) readAudioFromFile(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read audio file: %v", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}
