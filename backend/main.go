package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"math"
	"strings"
)

// ============ MODELS ============

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Sensor struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // "temperature", "humidity", "rain", "soil_moisture", "wind", "solar"
	Position Position `json:"position"`
	Active   bool     `json:"active"`
	Name     string   `json:"name"`
}

type ClimateData struct {
	Temperature    float64   `json:"temperature"`     // °C
	Humidity       float64   `json:"humidity"`        // %
	RainIntensity  float64   `json:"rain_intensity"`  // mm/h
	WindSpeed      float64   `json:"wind_speed"`      // km/h
	SolarRadiation float64   `json:"solar_radiation"` // W/m²
	Timestamp      time.Time `json:"timestamp"`
}

type SensorReading struct {
	SensorID  string      `json:"sensor_id"`
	Type      string      `json:"type"`
	Value     float64     `json:"value"`
	Unit      string      `json:"unit"`
	Position  Position    `json:"position"`
	Climate   ClimateData `json:"climate_context"`
	Timestamp time.Time   `json:"timestamp"`
}

// ============ CLIMATE SIMULATOR ============

type ClimateSimulator struct {
	current       ClimateData
	season        string
	baseTemp      float64
	timeOfDay     float64
	dayOfYear     int
	running       bool
	mu            sync.RWMutex
	subscribers   []chan ClimateData
}

func NewClimateSimulator() *ClimateSimulator {
	return &ClimateSimulator{
		season:    "spring",
		baseTemp:  22.0,
		timeOfDay: 12.0,
		dayOfYear: 80,
		current: ClimateData{
			Temperature:    22.0,
			Humidity:       65.0,
			RainIntensity:  0.0,
			WindSpeed:      10.0,
			SolarRadiation: 800.0,
			Timestamp:      time.Now(),
		},
		subscribers: make([]chan ClimateData, 0),
	}
}

func (cs *ClimateSimulator) Update() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.timeOfDay += 0.02 // Incrementa o tempo do dia
	if cs.timeOfDay >= 24 {
		cs.timeOfDay = 0
		cs.dayOfYear++
	}

	// Ciclo diário de temperatura
	dailyVariation := 8.0 * math.Cos((cs.timeOfDay-14.0)*3.14159/12.0)
	seasonalVariation := 10.0 * math.Sin(float64(cs.dayOfYear)*2*3.14159/365.0)
	noise := (randomFloat() - 0.5) * 2.0

	cs.current.Temperature = cs.baseTemp + dailyVariation + seasonalVariation + noise

	// Umidade inversamente proporcional à temperatura
	cs.current.Humidity = 70.0 - (cs.current.Temperature-cs.baseTemp)*2.0 + noise*5.0
	if cs.current.Humidity < 20 {
		cs.current.Humidity = 20
	}
	if cs.current.Humidity > 95 {
		cs.current.Humidity = 95
	}

	// Radiação solar baseada na hora
	if cs.timeOfDay > 6 && cs.timeOfDay < 18 {
		solarAngle := math.Sin((cs.timeOfDay - 6) * 3.14159 / 12)
		cs.current.SolarRadiation = 1000.0 * solarAngle
	} else {
		cs.current.SolarRadiation = 0
	}

	// Chuva aleatória
	rainProb := 0.001
	switch cs.season {
	case "summer":
		rainProb = 0.001
	case "winter":
		rainProb = 0.005
	default:
		rainProb = 0.003
	}

	if randomFloat() < rainProb && cs.current.RainIntensity == 0 {
		cs.current.RainIntensity = randomFloat() * 25.0
	} else if cs.current.RainIntensity > 0 {
		cs.current.RainIntensity -= 2.0
		if cs.current.RainIntensity < 0 {
			cs.current.RainIntensity = 0
		}
	}

	// Vento
	cs.current.WindSpeed = 5.0 + randomFloat()*15.0 + cs.current.RainIntensity*0.5
	cs.current.Timestamp = time.Now()

	// Notifica subscribers
	for _, ch := range cs.subscribers {
		select {
		case ch <- cs.current:
		default:
		}
	}
}

func (cs *ClimateSimulator) GetCurrent() ClimateData {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.current
}

func (cs *ClimateSimulator) SetSeason(season string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.season = season
	switch season {
	case "summer":
		cs.baseTemp = 28.0
	case "winter":
		cs.baseTemp = 15.0
	default:
		cs.baseTemp = 22.0
	}
}

func (cs *ClimateSimulator) Subscribe() chan ClimateData {
	ch := make(chan ClimateData, 10)
	cs.mu.Lock()
	cs.subscribers = append(cs.subscribers, ch)
	cs.mu.Unlock()
	return ch
}

func (cs *ClimateSimulator) Start() {
	cs.running = true
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for cs.running {
			<-ticker.C
			cs.Update()
		}
	}()
}

func (cs *ClimateSimulator) ForceRain() {
	cs.mu.Lock()
	cs.current.RainIntensity = 25.0 + randomFloat()*10.0
	cs.mu.Unlock()
}

// ============ SENSOR MANAGER ============

type SensorManager struct {
	sensors    map[string]*Sensor
	mu         sync.RWMutex
	climateSim *ClimateSimulator
	readings   chan SensorReading
	mqttClient mqtt.Client
}

func NewSensorManager(climate *ClimateSimulator, mqttClient mqtt.Client) *SensorManager {
	return &SensorManager{
		sensors:    make(map[string]*Sensor),
		climateSim: climate,
		readings:   make(chan SensorReading, 100),
		mqttClient: mqttClient,
	}
}

func (sm *SensorManager) AddSensor(sensorType string, pos Position, name string) *Sensor {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sensor := &Sensor{
		ID:       uuid.New().String(),
		Type:     sensorType,
		Position: pos,
		Active:   false,
		Name:     name,
	}
	sm.sensors[sensor.ID] = sensor
	return sensor
}

func (sm *SensorManager) GetSensor(id string) (*Sensor, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sensors[id]
	return s, ok
}

func (sm *SensorManager) GetAllSensors() []*Sensor {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sensors := make([]*Sensor, 0, len(sm.sensors))
	for _, s := range sm.sensors {
		sensors = append(sensors, s)
	}
	return sensors
}

func (sm *SensorManager) MoveSensor(id string, pos Position) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sensors[id]; ok {
		s.Position = pos
		return true
	}
	return false
}

func (sm *SensorManager) ToggleSensor(id string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sensors[id]; ok {
		s.Active = !s.Active
		return s.Active
	}
	return false
}

func (sm *SensorManager) RemoveSensor(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sensors, id)
}

func (sm *SensorManager) GenerateReading(sensor *Sensor, climate ClimateData) SensorReading {
	reading := SensorReading{
		SensorID:  sensor.ID,
		Type:      sensor.Type,
		Position:  sensor.Position,
		Climate:   climate,
		Timestamp: time.Now(),
	}

	// Altura afeta temperatura (-0.65°C/100m)
	altitudeEffect := sensor.Position.Y * -0.0065

	switch sensor.Type {
	case "temperature":
		reading.Value = climate.Temperature + altitudeEffect + (randomFloat()-0.5)
		reading.Unit = "°C"

	case "humidity":
		heightFactor := 1.0 - (sensor.Position.Y * 0.001)
		reading.Value = climate.Humidity*heightFactor + (randomFloat()-0.5)*2.0
		reading.Unit = "%"

	case "rain":
		reading.Value = climate.RainIntensity
		reading.Unit = "mm/h"

	case "soil_moisture":
		baseMoisture := 30.0
		rainEffect := climate.RainIntensity * 2.0
		depthFactor := 1.0 + math.Exp(-sensor.Position.Y/2.0)
		evaporation := climate.SolarRadiation / 1000.0 * 5.0
		reading.Value = (baseMoisture+rainEffect)*depthFactor - evaporation + (randomFloat()-0.5)*3.0
		if reading.Value > 100 {
			reading.Value = 100
		}
		if reading.Value < 0 {
			reading.Value = 0
		}
		reading.Unit = "%"

	case "wind":
		heightFactor := math.Pow(1.0+sensor.Position.Y/10.0, 0.2)
		reading.Value = climate.WindSpeed*heightFactor + (randomFloat()-0.5)*2.0
		reading.Unit = "km/h"

	case "solar":
		reading.Value = climate.SolarRadiation * (1.0 - sensor.Position.Y*0.01)
		if reading.Value < 0 {
			reading.Value = 0
		}
		reading.Unit = "W/m²"
	}

	return reading
}

func (sm *SensorManager) Start() {
	climateCh := sm.climateSim.Subscribe()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case climate := <-climateCh:
				sm.mu.RLock()
				sensors := make([]*Sensor, 0, len(sm.sensors))
				for _, s := range sm.sensors {
					sensors = append(sensors, s)
				}
				sm.mu.RUnlock()

				for _, sensor := range sensors {
					if !sensor.Active {
						continue
					}

					reading := sm.GenerateReading(sensor, climate)

					// Publica no MQTT
					topic := "agro/sensors/" + sensor.ID + "/data"
					payload, _ := json.Marshal(reading)
					if sm.mqttClient != nil && sm.mqttClient.IsConnected() {
						sm.mqttClient.Publish(topic, 0, false, payload)
					}

					// Envia para canal interno
					select {
					case sm.readings <- reading:
					default:
					}
				}

			case <-ticker.C:
				// Backup ticker
			}
		}
	}()
}

// ============ HTTP SERVER ============

type Server struct {
	sensorManager *SensorManager
	climateSim    *ClimateSimulator
	upgrader      websocket.Upgrader
	clients       map[*websocket.Conn]bool
	clientsMu     sync.RWMutex
	broadcastMu   sync.Mutex 
	mqttClient    mqtt.Client
}

func NewServer(sm *SensorManager, cs *ClimateSimulator, mqttClient mqtt.Client) *Server {
	return &Server{
		sensorManager: sm,
		climateSim:    cs,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients:    make(map[*websocket.Conn]bool),
		mqttClient: mqttClient,
	}
}

func (s *Server) getSensors(w http.ResponseWriter, r *http.Request) {
	sensors := s.sensorManager.GetAllSensors()
	json.NewEncoder(w).Encode(sensors)
}

func (s *Server) createSensor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string   `json:"type"`
		Position Position `json:"position"`
		Name     string   `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sensor := s.sensorManager.AddSensor(req.Type, req.Position, req.Name)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sensor)
}

func (s *Server) moveSensor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var pos Position
	if err := json.NewDecoder(r.Body).Decode(&pos); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !s.sensorManager.MoveSensor(id, pos) {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) toggleSensor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	active := s.sensorManager.ToggleSensor(id)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"active": active,
	})
}

func (s *Server) deleteSensor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	s.sensorManager.RemoveSensor(id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getClimate(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.climateSim.GetCurrent())
}

func (s *Server) setSeason(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Season string `json:"season"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.climateSim.SetSeason(req.Season)
	json.NewEncoder(w).Encode(map[string]string{"season": req.Season})
}

func (s *Server) forceRain(w http.ResponseWriter, r *http.Request) {
	s.climateSim.ForceRain()
	json.NewEncoder(w).Encode(map[string]string{"status": "rain started"})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	// Envia dados iniciais
	initData := map[string]interface{}{
		"type":    "init",
		"sensors": s.sensorManager.GetAllSensors(),
		"climate": s.climateSim.GetCurrent(),
	}
	conn.WriteJSON(initData)

	// Loop de leitura (mantém conexão aberta)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) Broadcast(data interface{}) {
    s.clientsMu.RLock()
    clients := make([]*websocket.Conn, 0, len(s.clients))
    for client := range s.clients {
        clients = append(clients, client)
    }
    s.clientsMu.RUnlock()

    for _, client := range clients {
        // Cada write em sua própria goroutine com lock
        go func(c *websocket.Conn) {
            s.broadcastMu.Lock()
            defer s.broadcastMu.Unlock()
            
            err := c.WriteJSON(data)
            if err != nil {
                s.clientsMu.Lock()
                delete(s.clients, c)
                s.clientsMu.Unlock()
                c.Close()
            }
        }(client)
    }
}

func (s *Server) StartBroadcasting() {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		for range ticker.C {
			s.Broadcast(map[string]interface{}{
				"type":    "climate",
				"climate": s.climateSim.GetCurrent(),
			})
		}
	}()

	go func() {
		for reading := range s.sensorManager.readings {
			s.Broadcast(map[string]interface{}{
				"type":    "reading",
				"reading": reading,
			})
		}
	}()
}

func (s *Server) Routes() *mux.Router {
	r := mux.NewRouter()

	// API REST
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/sensors", s.getSensors).Methods("GET")
	api.HandleFunc("/sensors", s.createSensor).Methods("POST")
	api.HandleFunc("/sensors/{id}/move", s.moveSensor).Methods("PUT")
	api.HandleFunc("/sensors/{id}/toggle", s.toggleSensor).Methods("POST")
	api.HandleFunc("/sensors/{id}", s.deleteSensor).Methods("DELETE")
	api.HandleFunc("/climate", s.getClimate).Methods("GET")
	api.HandleFunc("/climate/season", s.setSeason).Methods("PUT")
	api.HandleFunc("/climate/rain", s.forceRain).Methods("POST")

	// WebSocket
	r.HandleFunc("/ws", s.handleWebSocket)

	// Static files com headers corretos
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Define MIME types manualmente
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}
		
		http.FileServer(http.Dir("../frontend/")).ServeHTTP(w, r)
	})))

	return r
}

// ============ UTILS ============




func randomFloat() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}


// ============ MAIN ============

func main() {
	// Configuração MQTT
	// IMPORTANTE: Ajuste a porta se necessário (1883 ou 1884)
	mqttOpts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1884")
	mqttOpts.SetClientID("agro-sim-backend")
	mqttClient := mqtt.NewClient(mqttOpts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println("MQTT connection failed:", token.Error())
		log.Println("Continuing without MQTT...")
		mqttClient = nil
	} else {
		log.Println("MQTT connected successfully")
	}

	// Inicializa simuladores
	climateSim := NewClimateSimulator()
	climateSim.Start()

	sensorManager := NewSensorManager(climateSim, mqttClient)
	sensorManager.Start()

	// Cria servidor HTTP/WebSocket
	server := NewServer(sensorManager, climateSim, mqttClient)
	server.StartBroadcasting()

	// Rotas
	r := server.Routes()

	log.Println("Server starting on :8080")
	log.Println("API: http://localhost:8080/api/")
	log.Println("WebSocket: ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", r))
}