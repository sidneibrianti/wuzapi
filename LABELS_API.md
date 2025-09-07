# API de Labels do WhatsApp

Este documento descreve os novos endpoints implementados para gerenciamento de labels do WhatsApp na WuzAPI.

## Endpoints Disponíveis

### 1. Listar Labels
**Endpoint:** `GET /labels/list`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Resposta:**
```json
{
  "success": true,
  "message": "Labels functionality requires WhatsApp Business API. This is a placeholder implementation.",
  "labels": []
}
```

### 2. Criar Label
**Endpoint:** `POST /labels/create`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Body:**
```json
{
  "name": "Importante",
  "color": 1
}
```

**Resposta:**
```json
{
  "success": true,
  "label_id": "label_1693936427_Importante",
  "name": "Importante",
  "color": 1,
  "message": "Label created successfully"
}
```

### 3. Editar Label
**Endpoint:** `PUT /labels/edit`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Body:**
```json
{
  "label_id": "label_1693936427_Importante",
  "name": "Muito Importante",
  "color": 2
}
```

**Resposta:**
```json
{
  "success": true,
  "message": "Label updated successfully"
}
```

### 4. Deletar Label
**Endpoint:** `DELETE /labels/delete`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Body:**
```json
{
  "label_id": "label_1693936427_Importante"
}
```

**Resposta:**
```json
{
  "success": true,
  "message": "Label deleted successfully"
}
```

### 5. Listar Chats com Label
**Endpoint:** `GET /labels/chats?label_id=LABEL_ID`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Resposta:**
```json
{
  "success": true,
  "message": "Getting labeled chats requires app state synchronization. This is a placeholder implementation.",
  "chats": []
}
```

### 6. Associar Chat a Label
**Endpoint:** `POST /labels/associate`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Body:**
```json
{
  "chat_jid": "5511999999999@s.whatsapp.net",
  "label_id": "label_1693936427_Importante"
}
```

**Resposta:**
```json
{
  "success": true,
  "message": "Chat labeled successfully"
}
```

### 7. Remover Label do Chat
**Endpoint:** `POST /labels/disassociate`
**Headers:** `Token: SEU_TOKEN_DE_USUARIO`

**Body:**
```json
{
  "chat_jid": "5511999999999@s.whatsapp.net",
  "label_id": "label_1693936427_Importante"
}
```

**Resposta:**
```json
{
  "success": true,
  "message": "Label removed from chat successfully"
}
```

## Códigos de Cores para Labels

As cores disponíveis para labels são:
- 0: Azul
- 1: Verde
- 2: Rosa
- 3: Amarelo
- 4: Laranja
- 5: Vermelho
- 6: Roxo
- 7: Azul Claro
- 8: Verde Claro

## Observações Importantes

1. **Implementação com AppState**: Os endpoints utilizam o sistema de AppState do WhatsApp para sincronização de labels entre dispositivos.

2. **WhatsApp Business**: A funcionalidade completa de labels está mais disponível em contas do WhatsApp Business.

3. **Sincronização**: As labels são sincronizadas através do WhatsApp Web Protocol usando patches de estado da aplicação.

4. **IDs Únicos**: Os IDs das labels são gerados automaticamente no formato `label_{timestamp}_{nome}`.

## Exemplos de Uso com cURL

### Criar uma label:
```bash
curl -X POST http://localhost:8080/labels/create \
  -H "Token: SEU_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{"name": "Clientes VIP", "color": 2}'
```

### Associar um chat a uma label:
```bash
curl -X POST http://localhost:8080/labels/associate \
  -H "Token: SEU_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_jid": "5511999999999@s.whatsapp.net",
    "label_id": "label_1693936427_Clientes_VIP"
  }'
```

### Listar labels:
```bash
curl -X GET http://localhost:8080/labels/list \
  -H "Token: SEU_TOKEN_AQUI"
```

## Tratamento de Erros

Todos os endpoints retornam códigos de status HTTP apropriados:
- 200: Sucesso
- 400: Dados inválidos (Bad Request)
- 401: Token inválido (Unauthorized)
- 404: Cliente não encontrado (Not Found)
- 500: Erro interno do servidor

Em caso de erro, a resposta terá o formato:
```json
{
  "Details": "Mensagem de erro detalhada"
}
```
