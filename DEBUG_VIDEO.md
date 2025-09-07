# Configuração de Debug para WuzAPI

## Como habilitar logs detalhados

### 1. Via variável de ambiente
Antes de iniciar o servidor, configure:
```bash
# Windows PowerShell
$env:LOG_LEVEL = "debug"
.\wuzapi.exe

# Windows CMD
set LOG_LEVEL=debug
wuzapi.exe

# Linux/Mac
export LOG_LEVEL=debug
./wuzapi
```

### 2. Via parâmetro de linha de comando (se suportado)
```bash
./wuzapi --log-level=debug
```

### 3. Logs importantes para debug de vídeo

Quando você executar um teste de vídeo, procure por estas mensagens nos logs:

**Logs de Sucesso:**
- `"Iniciando processamento de envio de status de vídeo"`
- `"Request de vídeo decodificado"`
- `"Cliente WhatsApp obtido com sucesso"`
- `"Iniciando processamento de vídeo"`
- `"Vídeo processado com sucesso"`
- `"Validando formato de vídeo"`
- `"Validando tamanho do vídeo"`
- `"Enviando status de vídeo para WhatsApp"`
- `"Iniciando upload do vídeo para WhatsApp"`
- `"Upload do vídeo concluído com sucesso"`
- `"Status de vídeo enviado com sucesso"`

**Logs de Erro (o que procurar):**
- `"Erro ao decodificar payload JSON"`
- `"Falha na autenticação do cliente WhatsApp"`
- `"Falha ao processar source de vídeo"`
- `"Formato de vídeo não suportado"`
- `"Vídeo muito grande - máximo 64MB"`
- `"Falha no upload do vídeo"`
- `"Falha ao enviar mensagem de status de vídeo"`

### 4. Como executar o teste

1. **Inicie o servidor com debug:**
   ```bash
   $env:LOG_LEVEL = "debug"
   .\wuzapi.exe
   ```

2. **Em outro terminal, execute o teste:**
   ```bash
   .\test_video_debug.ps1
   ```

3. **Observe os logs** no terminal do servidor para identificar onde está falhando.

### 5. Pontos de falha comuns

1. **Processamento de source:** Falha ao detectar tipo de source (URL, base64, file)
2. **Decodificação:** Erro ao decodificar base64 ou baixar URL
3. **Validação:** MIME type não suportado
4. **Upload:** Falha na comunicação com WhatsApp
5. **Envio:** Erro ao enviar a mensagem de status

### 6. Exemplo de log completo esperado

```
DEBUG Iniciando processamento de envio de status de vídeo
DEBUG Request de vídeo decodificado source=base64 caption="Teste" video_length=1234
DEBUG Cliente WhatsApp obtido com sucesso
INFO  Iniciando processamento de vídeo source=base64
DEBUG Processando vídeo como base64
DEBUG Decodificando base64 de vídeo has_data_prefix=true data_prefix="data:video/mp4;base64,AAAA..."
DEBUG Data URL de vídeo decodificada com sucesso content_type=video/mp4 filename=video_1699123456.mp4 data_size=2048
INFO  Vídeo processado com sucesso mime_type=video/mp4 video_size=2048
DEBUG Validando formato de vídeo mime_type=video/mp4
DEBUG MIME type de vídeo válido mime_type=video/mp4
DEBUG Validando tamanho do vídeo video_size_mb=0
INFO  Enviando status de vídeo para WhatsApp mime_type=video/mp4 video_size_mb=0
DEBUG Iniciando envio de status de vídeo video_size=2048 mime_type=video/mp4 caption="Teste"
DEBUG Iniciando upload do vídeo para WhatsApp
DEBUG Upload do vídeo concluído com sucesso url="https://..." direct_path="/..." file_length=2048
DEBUG Criando mensagem de vídeo
DEBUG Adicionando caption ao vídeo caption="Teste"
DEBUG Enviando mensagem de status de vídeo
INFO  Status de vídeo enviado com sucesso message_id="ABC123" timestamp=2023-11-05T10:30:00Z
```
