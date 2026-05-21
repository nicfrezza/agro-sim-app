
# 🌾 AgroSim - Simulador de Agricultura de Precisão IoT

Sistema completo de simulação virtual de agricultura de precisão com sensores IoT virtuais, clima dinâmico em tempo real e visualização 3D interativa.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Three.js](https://img.shields.io/badge/Three.js-r128-black?style=for-the-badge&logo=three.js&logoColor=white)
![MQTT](https://img.shields.io/badge/MQTT-5.0-660066?style=for-the-badge)
![WebSocket](https://img.shields.io/badge/WebSocket-Enabled-010101?style=for-the-badge)

---

## 📋 Sumário

- [Sobre](#sobre)
- [Funcionalidades](#funcionalidades)
- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Instalação](#instalação)
- [Uso](#uso)
- [API](#api)
- [Estrutura](#estrutura)

---

## 🎯 Sobre

O **AgroSim** é uma plataforma educacional que simula um ambiente de agricultura de precisão, permitindo:

- Visualização 3D de um terreno agrícola
- Adição e manipulação de sensores IoT virtuais
- Simulação realista de clima (temperatura, umidade, chuva, vento, radiação solar)
- Comunicação em tempo real via MQTT e WebSocket
- Análise de dados dos sensores em dashboard

Ideal para aprender conceitos de IoT, MQTT, WebSocket e visualização 3D sem necessidade de hardware físico.

---

## ✨ Funcionalidades

### 🌍 Visualização 3D
- Terreno procedural com elevações naturais
- Sistema dia/noite dinâmico
- Rios e elementos paisagísticos
- Câmera orbital interativa

### 📡 Sensores Virtuais
| Tipo | Descrição | Unidade |
|------|-----------|---------|
| 🌡️ Temperatura | Mede temperatura ambiente | °C |
| 💧 Umidade do Ar | Umidade relativa | % |
| 🌧️ Pluviômetro | Intensidade de chuva | mm/h |
| 🌱 Umidade do Solo | Umidade no subsolo | % |
| 💨 Anemômetro | Velocidade do vento | km/h |
| ☀️ Radiação Solar | Irradiação solar | W/m² |

### 🌤️ Simulação Climática
- **Estações do ano**: Primavera, Verão, Outono, Inverno
- **Ciclo dia/noite**: Com base na radiação solar
- **Chuva dinâmica**: Probabilidade baseada na estação
- **Efeitos de altitude**: Temperatura varia com altura

### 📊 Comunicação em Tempo Real
- **MQTT**: Protocolo IoT padrão para publicação de dados
- **WebSocket**: Atualização instantânea no navegador
- **REST API**: Gerenciamento de sensores e configurações

---

## 🛠️ Tecnologias

### Backend
- **Go 1.21+**
  - `gorilla/mux` - Roteamento HTTP
  - `gorilla/websocket` - WebSocket server
  - `eclipse/paho.mqtt.golang` - Cliente MQTT
  - `google/uuid` - Geração de IDs

### Frontend
- **Three.js r128** - Renderização 3D
- **MQTT.js** - Cliente MQTT via WebSocket
- **HTML5/CSS3** - Interface responsiva

### Infraestrutura
- **Docker** - Containerização do broker MQTT
- **Eclipse Mosquitto** - Broker MQTT com suporte a WebSocket

---

## 🚀 Instalação

### Pré-requisitos
- [Go 1.21+](https://golang.org/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- Navegador moderno (Chrome, Firefox, Safari, Edge)

### Clone o repositório

# Instalar docker 

```bash
brew install --cask docker
```



```bash
git clone https://github.com/seu-usuario/agrosim.git
cd agrosim

docker run -d \
  --name mosquitto-broker \
  -p 1884:1883 \
  -p 9002:9001 \
  eclipse-mosquitto:2

  cd backend

```

# Baixa dependências
```go mod tidy```

# Inicia servidor
```go run main.go```

# Acesse a aplicação

Abra no navegador: http://localhost:8080

# 🎮 Uso
Primeiros Passos
- Adicione sensores: Clique nos botões do painel direito para adicionar sensores ao terreno
- Posicione: Arraste os sensores com o mouse para posições estratégicas
- Ative: Clique em "Ligar" no painel de sensores para começar a coleta de dados
- Observe: Veja os dados em tempo real no painel MQTT e na simulação 3D

# Simulação Climática
- Mude a estação no painel esquerdo para alterar temperatura base
- Clique em "Forçar Chuva" para simular precipitação
- Observe como a radiação solar afeta a cor do céu (dia/noite)

# 🔌 API Reference

REST Endpoints

Clima

| Método | Endpoint | Descrição |
|------|-----------|---------|
| GET | /api/climate | Obtém dados climáticos atuais|
| PUT | /api/climate/season | Altera estação do ano |
| POST| /api/climate/rain | Força chuva imediata |

Sensores

| Método | Endpoint | Descrição |
|------|-----------|---------|
| GET | /api/sensors | Lista todos os sensores |
| POST | /api/sensors | Cria novo sensor |
| PUT | /api/sensors/{id}/move | Move sensor de posição |
| POST | /api/sensors/{id}/toggle | Liga/desliga sensor |
| DELETE | /api/sensors/{id} | Remover sensor |

Exemplo: Criar sensor
``` curl -X POST http://localhost:8080/api/sensors \
  -H "Content-Type: application/json" \
  -d '{
    "type": "temperature",
    "position": {"x": 10, "y": 2, "z": 5},
    "name": "Temp_01"
  }'


```
# Estrutura do Projeto

```
agrosim/
├── 📂 backend/
│   ├── main.go              # Entry point e servidor HTTP
│   ├── go.mod               # Dependências Go
│   └── go.sum               # Checksums
│
├── 📂 frontend/
│   ├── index.html           # Interface principal
│   └── 📂 js/
│       └── app.js           # Lógica Three.js e comunicação
│
├── 📂 mosquitto/            # Configuração MQTT (opcional)
│   └── 📂 config/
│       └── mosquitto.conf   # Configurações do broker
│
└── README.md                # Este arquivo
```


