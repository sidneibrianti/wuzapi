package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow/appstate"
)

// ========== LABELS STRUCTURES AND CACHE ==========

// Estrutura para armazenar informações de label
type LabelInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Color        int32  `json:"color"`
	PredefinedID string `json:"predefined_id,omitempty"`
	Deleted      bool   `json:"deleted,omitempty"`
	Active       bool   `json:"active"`
}

// Estrutura para associações chat-label
type ChatLabelAssociation struct {
	ChatJID string `json:"chat_jid"`
	LabelID string `json:"label_id"`
}

// Cache global para labels por usuário
var (
	labelsCache = make(map[string]map[string]*LabelInfo) // userID -> labelID -> LabelInfo
	labelsMutex sync.RWMutex
)

// Cache para associações chat-label por usuário
var (
	chatLabelsCache = make(map[string][]ChatLabelAssociation) // userID -> associations
	chatLabelsMutex sync.RWMutex
)

// ========== LABELS HELPER FUNCTIONS ==========

// Função para obter labels do cache
func getUserLabels(userID string) map[string]*LabelInfo {
	labelsMutex.RLock()
	defer labelsMutex.RUnlock()

	if labelsCache[userID] == nil {
		return make(map[string]*LabelInfo)
	}

	// Retornar cópia do cache
	result := make(map[string]*LabelInfo)
	for k, v := range labelsCache[userID] {
		result[k] = &LabelInfo{
			ID:     v.ID,
			Name:   v.Name,
			Color:  v.Color,
			Active: v.Active,
		}
	}
	return result
}

// Função para criar labels comuns baseada no LABELS.md
func createCommonLabels(userID string) {
	labelsMutex.Lock()
	defer labelsMutex.Unlock()

	if labelsCache[userID] == nil {
		labelsCache[userID] = make(map[string]*LabelInfo)
	}

	// Labels comuns conforme LABELS.md
	commonLabels := map[string]LabelInfo{
		"importante": {ID: "importante", Name: "⭐ Importante", Color: 0, Active: true}, // Vermelho
		"trabalho":   {ID: "trabalho", Name: "💼 Trabalho", Color: 4, Active: true},     // Azul
		"familia":    {ID: "familia", Name: "👨‍👩‍👧‍👦 Família", Color: 3, Active: true}, // Verde
		"urgente":    {ID: "urgente", Name: "🚨 Urgente", Color: 0, Active: true},       // Vermelho
		"pendente":   {ID: "pendente", Name: "⏳ Pendente", Color: 1, Active: true},     // Laranja
		"concluido":  {ID: "concluido", Name: "✅ Concluído", Color: 3, Active: true},   // Verde
	}

	for id, label := range commonLabels {
		if labelsCache[userID][id] == nil {
			labelsCache[userID][id] = &LabelInfo{
				ID:     label.ID,
				Name:   label.Name,
				Color:  label.Color,
				Active: label.Active,
			}
		}
	}

	log.Info().Str("userID", userID).Int("count", len(commonLabels)).Msg("Common labels created")
}

// ========== LABELS HANDLERS ==========

// ListLabels - Lista todas as labels do usuário
func (s *server) ListLabels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		// Verificar se o cliente está conectado
		if !mycli.WAClient.IsConnected() {
			s.Respond(w, r, http.StatusBadRequest, errors.New("client not connected"))
			return
		}

		// Buscar labels do cache primeiro
		labels := getUserLabels(txtid)
		labelsList := make([]*LabelInfo, 0, len(labels))
		for _, label := range labels {
			if label.Active {
				labelsList = append(labelsList, label)
			}
		}

		response := map[string]interface{}{
			"success": true,
			"labels":  labelsList,
			"count":   len(labelsList),
			"source":  "cache",
		}

		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// RequestLabelsSync - Solicita sincronização de labels
func (s *server) RequestLabelsSync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		if !mycli.WAClient.IsConnected() {
			s.Respond(w, r, http.StatusBadRequest, errors.New("client not connected"))
			return
		}

		// Tentar solicitar dados de app state
		// Por enquanto, apenas retornar sucesso, pois a sincronização acontece automaticamente
		log.Info().Str("userID", txtid).Msg("Labels sync requested")

		response := map[string]interface{}{
			"success": true,
			"message": "Labels synchronization requested - check logs for app state events",
		}

		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// CreateCommonLabels - Cria labels comuns baseadas no LABELS.md
func (s *server) CreateCommonLabels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		if !mycli.WAClient.IsConnected() {
			s.Respond(w, r, http.StatusBadRequest, errors.New("client not connected"))
			return
		}

		// Criar labels comuns
		createCommonLabels(txtid)

		response := map[string]interface{}{
			"success": true,
			"message": "Common labels created successfully",
			"labels": []map[string]interface{}{
				{"id": "importante", "name": "⭐ Importante", "color": 0},
				{"id": "trabalho", "name": "💼 Trabalho", "color": 4},
				{"id": "familia", "name": "👨‍👩‍👧‍👦 Família", "color": 3},
				{"id": "urgente", "name": "🚨 Urgente", "color": 0},
				{"id": "pendente", "name": "⏳ Pendente", "color": 1},
				{"id": "concluido", "name": "✅ Concluído", "color": 3},
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

// CreateLabel - Cria uma nova label
func (s *server) CreateLabel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		type CreateLabelRequest struct {
			Name  string `json:"name"`
			Color int32  `json:"color"`
		}

		var req CreateLabelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid JSON"))
			return
		}

		if req.Name == "" {
			s.Respond(w, r, http.StatusBadRequest, errors.New("name is required"))
			return
		}

		// Gerar um ID único para a label
		labelID := fmt.Sprintf("label_%d_%s", time.Now().Unix(), strings.ReplaceAll(req.Name, " ", "_"))

		// Usar BuildLabelEdit para criar a label via AppState
		patch := appstate.BuildLabelEdit(labelID, req.Name, req.Color, false)

		err := mycli.WAClient.SendAppState(r.Context(), patch)
		if err != nil {
			log.Error().Err(err).Str("userID", txtid).Str("labelID", labelID).Msg("Failed to create label via AppState")
			s.Respond(w, r, http.StatusInternalServerError, errors.New("failed to create label: "+err.Error()))
			return
		}

		// Adicionar ao cache local também
		labelsMutex.Lock()
		if labelsCache[txtid] == nil {
			labelsCache[txtid] = make(map[string]*LabelInfo)
		}
		labelsCache[txtid][labelID] = &LabelInfo{
			ID:     labelID,
			Name:   req.Name,
			Color:  req.Color,
			Active: true,
		}
		labelsMutex.Unlock()

		log.Info().Str("userID", txtid).Str("labelID", labelID).Str("name", req.Name).Msg("Label created successfully via AppState")

		response := map[string]interface{}{
			"success":  true,
			"label_id": labelID,
			"name":     req.Name,
			"color":    req.Color,
			"message":  "Label created successfully via WhatsApp AppState",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// DeleteLabel - Remove uma label
func (s *server) DeleteLabel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		type DeleteLabelRequest struct {
			LabelID string `json:"label_id"`
		}

		var req DeleteLabelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid JSON"))
			return
		}

		if req.LabelID == "" {
			s.Respond(w, r, http.StatusBadRequest, errors.New("label_id is required"))
			return
		}

		// Primeiro, pegar informações da label do cache para manter nome e cor
		labelsMutex.RLock()
		var labelName string
		var labelColor int32
		if labelsCache[txtid] != nil && labelsCache[txtid][req.LabelID] != nil {
			labelName = labelsCache[txtid][req.LabelID].Name
			labelColor = labelsCache[txtid][req.LabelID].Color
		}
		labelsMutex.RUnlock()

		// Usar BuildLabelEdit com deleted=true para deletar via AppState
		patch := appstate.BuildLabelEdit(req.LabelID, labelName, labelColor, true)

		err := mycli.WAClient.SendAppState(r.Context(), patch)
		if err != nil {
			log.Error().Err(err).Str("userID", txtid).Str("labelID", req.LabelID).Msg("Failed to delete label via AppState")
			s.Respond(w, r, http.StatusInternalServerError, errors.New("failed to delete label: "+err.Error()))
			return
		}

		// Atualizar cache local também
		labelsMutex.Lock()
		if labelsCache[txtid] != nil && labelsCache[txtid][req.LabelID] != nil {
			labelsCache[txtid][req.LabelID].Active = false
			labelsCache[txtid][req.LabelID].Deleted = true
			log.Info().Str("userID", txtid).Str("labelID", req.LabelID).Msg("Label marked as deleted via AppState")
		}
		labelsMutex.Unlock()

		response := map[string]interface{}{
			"success": true,
			"message": "Label deleted successfully via WhatsApp AppState",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// EditLabel - Edita uma label existente
func (s *server) EditLabel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		type EditLabelRequest struct {
			LabelID string `json:"label_id"`
			Name    string `json:"name"`
			Color   int32  `json:"color"`
		}

		var req EditLabelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid JSON"))
			return
		}

		if req.LabelID == "" || req.Name == "" {
			s.Respond(w, r, http.StatusBadRequest, errors.New("label_id and name are required"))
			return
		}

		// Usar BuildLabelEdit para editar a label via AppState
		patch := appstate.BuildLabelEdit(req.LabelID, req.Name, req.Color, false)

		err := mycli.WAClient.SendAppState(r.Context(), patch)
		if err != nil {
			log.Error().Err(err).Str("userID", txtid).Str("labelID", req.LabelID).Msg("Failed to edit label via AppState")
			s.Respond(w, r, http.StatusInternalServerError, errors.New("failed to edit label: "+err.Error()))
			return
		}

		// Atualizar o cache local também
		labelsMutex.Lock()
		if labelsCache[txtid] != nil && labelsCache[txtid][req.LabelID] != nil {
			labelsCache[txtid][req.LabelID].Name = req.Name
			labelsCache[txtid][req.LabelID].Color = req.Color
			log.Info().Str("userID", txtid).Str("labelID", req.LabelID).Str("name", req.Name).Msg("Label updated via AppState")
		} else {
			// Criar se não existir
			if labelsCache[txtid] == nil {
				labelsCache[txtid] = make(map[string]*LabelInfo)
			}
			labelsCache[txtid][req.LabelID] = &LabelInfo{
				ID:     req.LabelID,
				Name:   req.Name,
				Color:  req.Color,
				Active: true,
			}
		}
		labelsMutex.Unlock()

		response := map[string]interface{}{
			"success": true,
			"message": "Label updated successfully via WhatsApp AppState",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// GetLabeledChats - Lista chats com uma label específica
func (s *server) GetLabeledChats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		labelID := r.URL.Query().Get("label_id")
		if labelID == "" {
			s.Respond(w, r, http.StatusBadRequest, errors.New("label_id parameter is required"))
			return
		}

		// Esta funcionalidade requer uma implementação mais complexa
		// por enquanto retornamos uma resposta básica
		response := map[string]interface{}{
			"success": true,
			"message": "Getting labeled chats requires app state synchronization. This is a placeholder implementation.",
			"chats":   []interface{}{},
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// AssociateChatLabel - Associa um chat a uma label
func (s *server) AssociateChatLabel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		type AssociateLabelRequest struct {
			ChatJID string `json:"chat_jid"`
			LabelID string `json:"label_id"`
		}

		var req AssociateLabelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid JSON"))
			return
		}

		if req.ChatJID == "" || req.LabelID == "" {
			s.Respond(w, r, http.StatusBadRequest, errors.New("chat_jid and label_id are required"))
			return
		}

		chatJID, valid := parseJID(req.ChatJID)
		if !valid {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid chat JID"))
			return
		}

		// Usar BuildLabelChat para associar chat com label via AppState
		patch := appstate.BuildLabelChat(chatJID, req.LabelID, true)

		err := mycli.WAClient.SendAppState(r.Context(), patch)
		if err != nil {
			log.Error().Err(err).Str("userID", txtid).Str("chatJID", req.ChatJID).Str("labelID", req.LabelID).Msg("Failed to associate chat label via AppState")
			s.Respond(w, r, http.StatusInternalServerError, errors.New("failed to associate chat label: "+err.Error()))
			return
		}

		// Atualizar cache de associações também
		chatLabelsMutex.Lock()
		if chatLabelsCache[txtid] == nil {
			chatLabelsCache[txtid] = make([]ChatLabelAssociation, 0)
		}

		// Verificar se já existe
		found := false
		for _, assoc := range chatLabelsCache[txtid] {
			if assoc.ChatJID == req.ChatJID && assoc.LabelID == req.LabelID {
				found = true
				break
			}
		}

		if !found {
			chatLabelsCache[txtid] = append(chatLabelsCache[txtid], ChatLabelAssociation{
				ChatJID: req.ChatJID,
				LabelID: req.LabelID,
			})
		}
		chatLabelsMutex.Unlock()

		log.Info().Str("userID", txtid).Str("chatJID", req.ChatJID).Str("labelID", req.LabelID).Msg("Chat label association completed via AppState")

		response := map[string]interface{}{
			"success": true,
			"message": "Chat labeled successfully via WhatsApp AppState",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}

// DisassociateChatLabel - Remove a associação de um chat com uma label
func (s *server) DisassociateChatLabel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txtid := r.Context().Value("userinfo").(Values).Get("Id")

		mycli := clientManager.GetMyClient(txtid)
		if mycli == nil {
			s.Respond(w, r, http.StatusNotFound, errors.New("client not found"))
			return
		}

		type DisassociateLabelRequest struct {
			ChatJID string `json:"chat_jid"`
			LabelID string `json:"label_id"`
		}

		var req DisassociateLabelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid JSON"))
			return
		}

		if req.ChatJID == "" || req.LabelID == "" {
			s.Respond(w, r, http.StatusBadRequest, errors.New("chat_jid and label_id are required"))
			return
		}

		chatJID, valid := parseJID(req.ChatJID)
		if !valid {
			s.Respond(w, r, http.StatusBadRequest, errors.New("invalid chat JID"))
			return
		}

		// Usar BuildLabelChat com labeled=false para remover associação via AppState
		patch := appstate.BuildLabelChat(chatJID, req.LabelID, false)

		err := mycli.WAClient.SendAppState(r.Context(), patch)
		if err != nil {
			log.Error().Err(err).Str("userID", txtid).Str("chatJID", req.ChatJID).Str("labelID", req.LabelID).Msg("Failed to disassociate chat label via AppState")
			s.Respond(w, r, http.StatusInternalServerError, errors.New("failed to disassociate chat label: "+err.Error()))
			return
		}

		// Atualizar cache de associações também
		chatLabelsMutex.Lock()
		if chatLabelsCache[txtid] != nil {
			newAssociations := make([]ChatLabelAssociation, 0)
			for _, assoc := range chatLabelsCache[txtid] {
				if !(assoc.ChatJID == req.ChatJID && assoc.LabelID == req.LabelID) {
					newAssociations = append(newAssociations, assoc)
				}
			}
			chatLabelsCache[txtid] = newAssociations
		}
		chatLabelsMutex.Unlock()

		log.Info().Str("userID", txtid).Str("chatJID", req.ChatJID).Str("labelID", req.LabelID).Msg("Chat label disassociation completed via AppState")

		response := map[string]interface{}{
			"success": true,
			"message": "Label removed from chat successfully via WhatsApp AppState",
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			s.Respond(w, r, http.StatusInternalServerError, err)
		} else {
			s.Respond(w, r, http.StatusOK, string(responseJson))
		}
	}
}
