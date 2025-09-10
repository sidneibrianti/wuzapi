package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

// ========== STATUS HANDLERS ==========
// Note: GetStatus handler is in handlers.go for comprehensive user info

// Send text status with formatting options
func (s *server) StatusSendText() http.HandlerFunc {
	type statusTextRequest struct {
		Text            string `json:"text" validate:"required,max=650"`
		BackgroundColor *int64 `json:"background_color,omitempty"` // ARGB decimal
		TextColor       *int64 `json:"text_color,omitempty"`       // ARGB decimal
		Font            *int32 `json:"font,omitempty"`             // 0-10 (fonts disponíveis)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req statusTextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("could not decode Payload"))
			return
		}

		// Validar font se fornecida
		if req.Font != nil && !s.isValidFont(*req.Font) {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid font - must be between 0-10"))
			return
		}

		mycli, err := s.getWAClient(r)
		if err != nil {
			s.Respond(w, r, http.StatusUnauthorized, errors.New("authentication failed"))
			return
		}

		// Enviar status de texto formatado
		messageInfo, err := s.sendFormattedTextStatus(mycli, req.Text, req.BackgroundColor, req.TextColor, req.Font)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		response := map[string]interface{}{
			"message_id": messageInfo.ID,
			"timestamp":  messageInfo.Timestamp.Format("2006-01-02T15:04:05Z"),
			"status":     "sent",
			"type":       "text_status",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// Send image status
func (s *server) StatusSendImage() http.HandlerFunc {
	type statusImageRequest struct {
		Image   string `json:"image" validate:"required"`
		Caption string `json:"caption,omitempty"`
		Source  string `json:"source,omitempty"` // "base64", "url", "file"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req statusImageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("could not decode Payload"))
			return
		}

		mycli, err := s.getWAClient(r)
		if err != nil {
			s.Respond(w, r, http.StatusUnauthorized, errors.New("authentication failed"))
			return
		}

		// Processar imagem baseado na fonte
		imageData, mimeType, err := s.processImageSource(req.Image, req.Source)
		if err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("failed to process image"))
			return
		}

		// Validar formato suportado
		if !s.isValidImageMimeType(mimeType) {
			s.Respond(w, r, http.StatusBadRequest, errors.New("unsupported image format"))
			return
		}

		// Enviar status com imagem
		messageInfo, err := s.sendImageStatus(mycli, imageData, mimeType, req.Caption)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		response := map[string]interface{}{
			"message_id": messageInfo.ID,
			"timestamp":  messageInfo.Timestamp.Format("2006-01-02T15:04:05Z"),
			"status":     "sent",
			"type":       "image_status",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// Send video status
func (s *server) StatusSendVideo() http.HandlerFunc {
	type statusVideoRequest struct {
		Video   string `json:"video" validate:"required"`
		Caption string `json:"caption,omitempty"`
		Source  string `json:"source,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info().Msg("Iniciando processamento de envio de status de vídeo")

		var req statusVideoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error().Err(err).Msg("Erro ao decodificar payload JSON")
			s.Respond(w, r, http.StatusBadRequest, errors.New("could not decode Payload"))
			return
		}

		log.Debug().
			Str("source", req.Source).
			Str("caption", req.Caption).
			Int("video_length", len(req.Video)).
			Msg("Request de vídeo decodificado")

		mycli, err := s.getWAClient(r)
		if err != nil {
			log.Error().Err(err).Msg("Falha na autenticação do cliente WhatsApp")
			s.Respond(w, r, http.StatusUnauthorized, errors.New("authentication failed"))
			return
		}

		log.Debug().Msg("Cliente WhatsApp obtido com sucesso")

		// Processar vídeo
		log.Info().Str("source", req.Source).Msg("Iniciando processamento de vídeo")
		videoData, mimeType, err := s.processVideoSource(req.Video, req.Source)
		if err != nil {
			log.Error().Err(err).Str("source", req.Source).Msg("Falha ao processar source de vídeo")
			s.Respond(w, r, http.StatusBadRequest, errors.New("failed to process video"))
			return
		}

		log.Info().
			Str("mime_type", mimeType).
			Int("video_size", len(videoData)).
			Msg("Vídeo processado com sucesso")

		// Validar formato
		log.Debug().Str("mime_type", mimeType).Msg("Validando formato de vídeo")
		if !s.isValidVideoMimeType(mimeType) {
			log.Error().Str("mime_type", mimeType).Msg("Formato de vídeo não suportado")
			s.Respond(w, r, http.StatusBadRequest, errors.New("unsupported video format"))
			return
		}

		// Validar tamanho do vídeo
		videoSizeMB := len(videoData) / (1024 * 1024)
		log.Debug().Int("video_size_mb", videoSizeMB).Msg("Validando tamanho do vídeo")
		if len(videoData) > 64*1024*1024 { // 64MB
			log.Error().
				Int("video_size_mb", videoSizeMB).
				Msg("Vídeo muito grande - máximo 64MB")
			s.Respond(w, r, http.StatusBadRequest, errors.New("video file too large - maximum size is 64MB"))
			return
		}

		// Enviar status com vídeo
		log.Info().
			Str("mime_type", mimeType).
			Int("video_size_mb", videoSizeMB).
			Msg("Enviando status de vídeo para WhatsApp")
		messageInfo, err := s.sendVideoStatus(mycli, videoData, mimeType, req.Caption)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao enviar status de vídeo")
			s.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		log.Info().
			Str("message_id", messageInfo.ID).
			Time("timestamp", messageInfo.Timestamp).
			Msg("Status de vídeo enviado com sucesso")

		response := map[string]interface{}{
			"message_id": messageInfo.ID,
			"timestamp":  messageInfo.Timestamp.Format("2006-01-02T15:04:05Z"),
			"status":     "sent",
			"type":       "video_status",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			log.Error().Err(err).Msg("Erro ao serializar response JSON")
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			log.Debug().Str("response", string(responseJson)).Msg("Response JSON criado")
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// Send audio status
func (s *server) StatusSendAudio() http.HandlerFunc {
	type statusAudioRequest struct {
		Audio  string `json:"audio" validate:"required"`
		Source string `json:"source,omitempty"` // "base64", "url", "file"
		PTT    bool   `json:"ptt,omitempty"`    // Push-to-talk (voice note)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req statusAudioRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("could not decode Payload"))
			return
		}

		mycli, err := s.getWAClient(r)
		if err != nil {
			s.Respond(w, r, http.StatusUnauthorized, errors.New("authentication failed"))
			return
		}

		// Processar áudio baseado na fonte
		audioData, mimeType, err := s.processAudioSource(req.Audio, req.Source)
		if err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("failed to process audio"))
			return
		}

		// Validar formato de áudio
		if !s.isValidAudioMimeType(mimeType) {
			s.Respond(w, r, http.StatusBadRequest, errors.New("unsupported audio format"))
			return
		}

		// Enviar status com áudio
		messageInfo, err := s.sendAudioStatus(mycli, audioData, mimeType, req.PTT)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
			return
		}

		response := map[string]interface{}{
			"message_id": messageInfo.ID,
			"timestamp":  messageInfo.Timestamp.Format("2006-01-02T15:04:05Z"),
			"status":     "sent",
			"type":       "audio_status",
			"ptt":        req.PTT,
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// Get status privacy settings
func (s *server) StatusPrivacy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mycli, err := s.getWAClient(r)
		if err != nil {
			s.Respond(w, r, http.StatusUnauthorized, errors.New("authentication failed"))
			return
		}

		// Get current connection status and user info
		isConnected := mycli.WAClient.IsConnected()
		var userJID string
		if mycli.WAClient.Store != nil && mycli.WAClient.Store.ID != nil {
			userJID = mycli.WAClient.Store.ID.String()
		}

		// Create real response with current status
		response := map[string]interface{}{
			"success":      true,
			"connected":    isConnected,
			"user_jid":     userJID,
			"privacy_note": "Status privacy settings are managed through WhatsApp mobile app",
			"available_settings": map[string]interface{}{
				"who_can_see_status": []string{
					"My contacts",
					"My contacts except...",
					"Only share with...",
				},
				"read_receipts": "Controlled by WhatsApp app settings",
				"last_seen":     "Controlled by WhatsApp app settings",
			},
			"api_limitations": []string{
				"Cannot modify privacy settings via API",
				"Cannot retrieve current privacy settings",
				"Status visibility follows WhatsApp app configuration",
			},
		}

		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// Note: Helper functions like getWAClient, sendFormattedTextStatus, sendImageStatus, etc.
// are defined in helpers.go and shared across all handlers
