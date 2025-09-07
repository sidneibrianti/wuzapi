package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vincent-petithory/dataurl"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"github.com/rs/zerolog/log"
)

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
	log.Debug().
		Int("video_size", len(videoData)).
		Str("mime_type", mimeType).
		Str("caption", caption).
		Msg("Iniciando envio de status de vídeo")

	ctx := context.Background()

	// Upload the video
	log.Debug().Msg("Iniciando upload do vídeo para WhatsApp")
	uploaded, err := mycli.WAClient.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		log.Error().Err(err).Msg("Falha no upload do vídeo")
		return nil, fmt.Errorf("upload failed: %v", err)
	}

	log.Debug().
		Str("url", uploaded.URL).
		Str("direct_path", uploaded.DirectPath).
		Uint64("file_length", uploaded.FileLength).
		Msg("Upload do vídeo concluído com sucesso")

	// Create video message
	log.Debug().Msg("Criando mensagem de vídeo")
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
		log.Debug().Str("caption", caption).Msg("Adicionando caption ao vídeo")
		videoMsg.Caption = proto.String(caption)
	}

	log.Debug().Msg("Enviando mensagem de status de vídeo")
	resp, err := mycli.WAClient.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		VideoMessage: videoMsg,
	})
	if err != nil {
		log.Error().Err(err).Msg("Falha ao enviar mensagem de status de vídeo")
		return nil, err
	}

	log.Info().
		Str("message_id", resp.ID).
		Time("timestamp", resp.Timestamp).
		Msg("Status de vídeo enviado com sucesso")

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

	log.Debug().Str("mime_type", mimeType).Msg("Validando MIME type de vídeo")

	for _, validType := range validTypes {
		if mimeType == validType {
			log.Debug().Str("mime_type", mimeType).Msg("MIME type de vídeo válido")
			return true
		}
	}

	log.Warn().Str("mime_type", mimeType).Msg("MIME type de vídeo inválido")
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
	log.Debug().
		Str("source_type", source).
		Str("video_length", fmt.Sprintf("%d", len(video))).
		Msg("Iniciando processamento de vídeo")

	if video == "" {
		log.Error().Msg("Dados de vídeo estão vazios")
		return nil, "", fmt.Errorf("video data cannot be empty")
	}

	var data []byte
	var mimeType string
	var err error

	switch source {
	case "base64", "":
		log.Debug().Msg("Processando vídeo como base64")
		data, mimeType, err = s.decodeBase64Video(video)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao decodificar base64 do vídeo")
			return nil, "", err
		}
	case "url":
		log.Debug().Str("url", video).Msg("Processando vídeo como URL")
		data, mimeType, err = s.downloadVideoFromURL(video)
		if err != nil {
			log.Error().Err(err).Str("url", video).Msg("Erro ao baixar vídeo da URL")
			return nil, "", err
		}
	case "file":
		log.Debug().Str("file_path", video).Msg("Processando vídeo como arquivo")
		data, mimeType, err = s.readVideoFromFile(video)
		if err != nil {
			log.Error().Err(err).Str("file_path", video).Msg("Erro ao ler arquivo de vídeo")
			return nil, "", err
		}
	default:
		log.Error().Str("source", source).Msg("Tipo de source não suportado")
		return nil, "", fmt.Errorf("unsupported source type: %s", source)
	}

	log.Debug().
		Str("mime_type", mimeType).
		Int("data_size", len(data)).
		Msg("Vídeo processado com sucesso")

	return data, mimeType, nil
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
	log.Debug().
		Bool("has_data_prefix", strings.HasPrefix(data, "data:video")).
		Str("data_prefix", data[:min(50, len(data))]).
		Msg("Decodificando base64 de vídeo")

	if strings.HasPrefix(data, "data:video") {
		log.Debug().Msg("Processando data URL de vídeo")
		dataURL, err := dataurl.DecodeString(data)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao decodificar data URL de vídeo")
			return nil, "", fmt.Errorf("could not decode base64 data URL: %v", err)
		}

		contentType := dataURL.MediaType.ContentType()
		log.Debug().
			Str("content_type", contentType).
			Int("data_size", len(dataURL.Data)).
			Msg("Data URL de vídeo decodificada com sucesso")

		return dataURL.Data, contentType, nil
	}

	log.Debug().Msg("Decodificando base64 puro de vídeo")
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao decodificar base64 puro de vídeo")
		return nil, "", fmt.Errorf("could not decode base64: %v", err)
	}

	mimeType := http.DetectContentType(decoded)
	log.Debug().
		Str("detected_mime_type", mimeType).
		Int("decoded_size", len(decoded)).
		Msg("Base64 de vídeo decodificado com sucesso")

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
	log.Debug().Str("url", url).Msg("Iniciando download de vídeo da URL")

	resp, err := http.Get(url)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Erro ao fazer request HTTP para vídeo")
		return nil, "", fmt.Errorf("failed to download video: %v", err)
	}
	defer resp.Body.Close()

	log.Debug().
		Int("status_code", resp.StatusCode).
		Str("content_type", resp.Header.Get("Content-Type")).
		Str("content_length", resp.Header.Get("Content-Length")).
		Msg("Response HTTP recebida para vídeo")

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("url", url).
			Msg("Status HTTP não é 200 OK para download de vídeo")
		return nil, "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Erro ao ler dados do vídeo da response")
		return nil, "", fmt.Errorf("failed to read video data: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
		log.Debug().Str("detected_content_type", contentType).Msg("Content-Type detectado automaticamente")
	}

	log.Debug().
		Str("content_type", contentType).
		Int("data_size", len(data)).
		Str("url", url).
		Msg("Vídeo baixado da URL com sucesso")

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
	log.Debug().Str("file_path", path).Msg("Lendo vídeo do arquivo")

	// Verificar se arquivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Error().Str("file_path", path).Msg("Arquivo de vídeo não existe")
		return nil, "", fmt.Errorf("video file does not exist: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Str("file_path", path).Msg("Erro ao ler arquivo de vídeo")
		return nil, "", fmt.Errorf("failed to read video file: %v", err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
		log.Debug().Str("detected_mime_type", mimeType).Msg("MIME type detectado automaticamente")
	}

	log.Debug().
		Str("file_path", path).
		Str("mime_type", mimeType).
		Int("file_size", len(data)).
		Msg("Arquivo de vídeo lido com sucesso")

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

// Helper functions for media processing

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isBase64 checks if a string is valid base64
func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// getExtensionFromMimeType returns file extension for common MIME types
func getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "video/mp4":
		return ".mp4"
	case "video/3gpp":
		return ".3gp"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "audio/aac":
		return ".aac"
	default:
		return ".bin"
	}
}
