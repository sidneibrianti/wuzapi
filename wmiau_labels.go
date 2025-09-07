package main

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow/types/events"
)

// processLabelEvents processa eventos relacionados a labels no event handler principal
func (mycli *MyClient) processLabelEvents(evt interface{}, postmap map[string]interface{}) int {
	switch evt := evt.(type) {
	case *events.AppState:
		// Processar eventos de labels via AppState
		handleLabelAppStateEvent(mycli.userID, evt)
		postmap["type"] = "AppState"
		return 1 // dowebhook = 1

	case *events.LabelAssociationChat:
		// Processar eventos específicos de associação chat-label
		log.Info().
			Str("userID", mycli.userID).
			Str("chatJID", evt.JID.String()).
			Str("labelID", evt.LabelID).
			Str("action", fmt.Sprintf("%+v", evt.Action)).
			Msg("Chat label association event received")

		// Atualizar cache local
		chatLabelsMutex.Lock()
		if chatLabelsCache[mycli.userID] == nil {
			chatLabelsCache[mycli.userID] = []ChatLabelAssociation{}
		}

		// Verificar se é labeled=true através da action
		isLabeled := evt.Action != nil && evt.Action.Labeled != nil && *evt.Action.Labeled

		if isLabeled {
			// Adicionar associação
			association := ChatLabelAssociation{
				ChatJID: evt.JID.String(),
				LabelID: evt.LabelID,
			}

			// Verificar se já existe
			found := false
			for _, existing := range chatLabelsCache[mycli.userID] {
				if existing.ChatJID == evt.JID.String() && existing.LabelID == evt.LabelID {
					found = true
					break
				}
			}

			if !found {
				chatLabelsCache[mycli.userID] = append(chatLabelsCache[mycli.userID], association)
			}
		} else {
			// Remover associação
			for i, existing := range chatLabelsCache[mycli.userID] {
				if existing.ChatJID == evt.JID.String() && existing.LabelID == evt.LabelID {
					chatLabelsCache[mycli.userID] = append(chatLabelsCache[mycli.userID][:i], chatLabelsCache[mycli.userID][i+1:]...)
					break
				}
			}
		}
		chatLabelsMutex.Unlock()

		postmap["type"] = "LabelAssociationChat"
		return 1 // dowebhook = 1

	case *events.LabelEdit:
		// Processar eventos de edição de labels
		log.Info().
			Str("userID", mycli.userID).
			Str("labelID", evt.LabelID).
			Str("action", fmt.Sprintf("%+v", evt.Action)).
			Msg("Label edit event received")

		// Atualizar cache local
		labelsMutex.Lock()
		if labelsCache[mycli.userID] == nil {
			labelsCache[mycli.userID] = make(map[string]*LabelInfo)
		}

		// Extrair informações da action
		labelInfo := &LabelInfo{
			ID:     evt.LabelID,
			Active: true,
		}

		if evt.Action != nil {
			if evt.Action.Name != nil {
				labelInfo.Name = *evt.Action.Name
			}
			if evt.Action.Color != nil {
				labelInfo.Color = *evt.Action.Color
			}
			if evt.Action.PredefinedID != nil {
				labelInfo.PredefinedID = fmt.Sprintf("%d", *evt.Action.PredefinedID)
			}
			if evt.Action.Deleted != nil {
				labelInfo.Deleted = *evt.Action.Deleted
				labelInfo.Active = !*evt.Action.Deleted
			}
		}

		labelsCache[mycli.userID][evt.LabelID] = labelInfo
		labelsMutex.Unlock()

		postmap["type"] = "LabelEdit"
		return 1 // dowebhook = 1
	}

	return 0 // não é um evento de label
}

// handleLabelAppStateEvent processa eventos de AppState relacionados a labels
func handleLabelAppStateEvent(userID string, evt *events.AppState) {
	labelsMutex.Lock()
	defer labelsMutex.Unlock()

	// Inicializar caches se necessário
	if labelsCache[userID] == nil {
		labelsCache[userID] = make(map[string]*LabelInfo)
	}

	chatLabelsMutex.Lock()
	defer chatLabelsMutex.Unlock()
	if chatLabelsCache[userID] == nil {
		chatLabelsCache[userID] = []ChatLabelAssociation{}
	}

	log.Info().Str("userID", userID).
		Str("index", fmt.Sprintf("%+v", evt.Index)).
		Msg("Processing label AppState event")

	// Processar index do evento AppState
	if len(evt.Index) >= 2 {
		switch evt.Index[0] {
		case "label_edit":
			// Processar mudanças em labels
			labelID := evt.Index[1]

			log.Info().Str("userID", userID).Str("labelID", labelID).Msg("Processing label edit event")

		case "label_jid":
			// Processar associações chat-label
			if len(evt.Index) >= 3 {
				labelID := evt.Index[1]
				chatJID := evt.Index[2]

				// Verificar se é uma associação (labeled:true) ou desassociação
				actionStr := fmt.Sprintf("%+v", evt.SyncActionValue)
				isLabeled := strings.Contains(actionStr, "labeled:true")

				if isLabeled {
					// Adicionar associação
					association := ChatLabelAssociation{
						ChatJID: chatJID,
						LabelID: labelID,
					}

					// Verificar se já existe
					found := false
					for _, existing := range chatLabelsCache[userID] {
						if existing.ChatJID == chatJID && existing.LabelID == labelID {
							found = true
							break
						}
					}

					if !found {
						chatLabelsCache[userID] = append(chatLabelsCache[userID], association)
					}

					log.Info().
						Str("userID", userID).
						Str("chatJID", chatJID).
						Str("labelID", labelID).
						Msg("Chat label association added via AppState")
				} else {
					// Remover associação
					for i, existing := range chatLabelsCache[userID] {
						if existing.ChatJID == chatJID && existing.LabelID == labelID {
							chatLabelsCache[userID] = append(chatLabelsCache[userID][:i], chatLabelsCache[userID][i+1:]...)
							break
						}
					}

					log.Info().
						Str("userID", userID).
						Str("chatJID", chatJID).
						Str("labelID", labelID).
						Msg("Chat label association removed via AppState")
				}
			}
		}
	}
}
