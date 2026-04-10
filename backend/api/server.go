package api

// esse script é responsável por configurar o servidor HTTP, definir as rotas para a API REST e lidar com conexões WebSocket para enviar atualizações em tempo real aos clientes. Ele também integra o SensorManager e o ClimateSimulator para fornecer dados atualizados sobre os sensores e o clima.

import (
    "encoding/json"
    "log"
    "net/http"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "agro-sim/models"
    "agro-sim/simulation"
    mqtt "github.com/eclipse/paho.mqtt.golang"
)


type Server struct {
	sensorManager *simulation.SensorManager // Gerenciador de sensores para lidar com as operações relacionadas aos sensores
	climateSim *simulation.ClimateSimulator // Simulador de clima para acessar os dados climáticos atuais
	upgrader websocket.Upgrader // Upgrader para lidar com conexões WebSocket
	clientes map[*websocket.Conn]bool // Mapa para rastrear os clientes conectados via WebSocket
	mqttClient mqtt.Client // Cliente MQTT para publicar mensagens
}

func NewServicer(sm *simulation.SensorManager, cs *simulation.ClimateSimulator) *Server {
	return &Server{ // o & é usado para retornar um ponteiro para a struct Server, permitindo que as funções do servidor modifiquem o estado do servidor e acessem os campos de forma eficiente.
		sensorManager: sm, // Atribui o SensorManager ao campo sensorManager do servidor
		climateSim: cs, // Atribui o ClimateSimulator ao campo climateSim do servidor
		upgrader: websocket.Upgrader{ // Configura o upgrader para permitir conexões WebSocket de qualquer origem
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clientes: make(map[*websocket.Conn]bool), // Inicializa o mapa de clientes conectados
	}
}

func (s *Server) SetupMQTT() {
	opts := mqtt;NewClientOptions().AddBroker("tcp://localhost:1883") 
	s.mqttClient = mqtt.NewClient(opts)
	if token := s.mqttCliente.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Erro ao conectar ao broker MQTT: %v", token.Error()) // Loga um erro fatal se a conexão ao broker MQTT falhar
	}
}



func (s *Server) Routes() *mux.Router {
	r := mux.NewRouter()

	//rest API 
	r.HandleFunc("/api/sensors", s.getSensors).Methods("GET") // Rota para obter a lista de sensores
	r.HandleFunc("/api/sensors", s.createSensor).Methods("POST") // Rota para criar um novo sensor
	r.HandleFunc("/api/sensors/{id}/move", s.moveSensor).Methods("PUT") // Rota para mover um sensor existente
	r.HandleFunc("/api/sensors/{id}/toggle", s.toggleSensor).Methods("POST") // Rota para alternar o estado ativo de um sensor
	r.HandleFunc("/api/sensors/{id}", s.deleteSensor).Methods("DELETE") // Rota para excluir um sensor existente
	r.HandleFunc("/api/climate", s.getClimate).Methods("GET") // Rota para obter os dados climáticos atuais
	r.HandleFunc("/api/climate/season", s.setSeason).Methods("POST") // Rota para definir a estação atual


	r.HandleFunc("/ws", .handleWebSocket) // Rota para lidar com conexões WebSocket

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend/")))

	return r

}


func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erro ao atualizar para WebSocket: %v", err)
		return
	}
	defer conn.Close() // Garante que a conexão WebSocket seja fechada quando a função retornar

	s.clientes[conn] = true

	sensors := s.sensorManager.GetAll() // Obtém a lista de todos os sensores do SensorManager
	conn.WriteJSON(map[string]interface{}{
		"type": "init", // Tipo da mensagem para indicar que é uma mensagem de inicialização
		"sensors": sensors, // Inclui a lista de sensores na mensagem JSON enviada para o cliente
		"climate": s.climateSim.GetCurrent(), // Inclui os dados climáticos atuais na mensagem JSON enviada para o cliente
	}) // Envia a lista de sensores para o cliente conectado via WebSocket

	// loop de leitura 
	for {
		_, _, err := conn.ReadMessage() // Lê mensagens do cliente, mas não faz nada com elas (pode ser expandido para lidar com mensagens do cliente no futuro)
		if err != nil {
			log.Printf("Erro ao ler mensagem do WebSocket: %v", err)
			delete(s.clientes, conn) // Remove o cliente do mapa de clientes conectados se ocorrer um erro ao ler a mensagem
			break
		}
	}


	// Broad cast para todos os clientes WebSocket conectados
	func (s *Server) Broadcast(data interface{}) {
		for cliente := range s.clientes { // Itera sobre todos os clientes conectados
			err := cliente.WriteJSON(data) // Envia os dados como JSON para cada cliente
			if err != nil {
				log.Printf("Erro ao enviar mensagem para cliente WebSocket: %v", err)
				cliente.Close() // Fecha a conexão do cliente se ocorrer um erro ao enviar a mensagem
				delete(s.clientes, cliente) // Remove o cliente do mapa de clientes conectados
			}
		}
	}	

