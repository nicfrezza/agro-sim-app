package simulation 

// aqui são criados os sensores, suas leituras e a lógica de simulação de clima para gerar dados realistas para os sensores.

import {
	"math"
	"math/rand"
	"time"
	"iot-hub/models" // Importing the models package to use the defined types
}

type SensorManager struct {
	sensors map[string]*models.Sensor // Mapa para armazenar os sensores, onde a chave é o ID do sensor como string
	climateSom *ClimateSimulator // Faz a referência ao simulador de clima para acessar os dados climáticos atuais
	readins chan models.SensorReading // Canal para enviar as leituras dos sensores
	mqttPublish func(topic string, payload []byte) // Função para publicar mensagens MQTT
}


func NewSensorManager(climateSim *ClimateSimulator, mqttPub func(string, []byte)) *SensorManager {
	return *SensorManager{
		sensors: make(map[string]*models.Sensor), // Inicializa o mapa de sensores
		climateSim: climateSim, // Atribui o simulador de clima ao SensorManager
		readins: make(chan models.SensorReading, 100), // Cria um canal para leituras de sensores com buffer de 100
		mqttPublish: mqttPub, // Atribui a função de publicação MQTT ao SensorManager
	}
}


func (sm *SensorManager) AddSensor(sensorType string, pos models.Position, name string) *models.Sensor {
	sensor := models.NewSensor(sensorType, pos, name) // Cria um novo sensor usando a função NewSensor
	sm.sensors[sensor.ID.String()] = sensor // Adiciona o sensor ao mapa de sensores usando seu ID como chave
	return sensor // Retorna o sensor criado
}

func (sm *SensorManager) MoveSensor(id string, newPos models.Position) bool{
	if s, ok := sm.sensors[id]; ok {
		s.Position = newPos // Atualiza a posição do sensor
		return true // Retorna true se o sensor foi encontrado e movido com sucesso
	}
	return false // Retorna false se o sensor com o ID fornecido não foi encontrado
}

// faz a leitura do sensor e publica os dados no canal de leituras e no MQTT
func (sm *SensorManager) ToggleSensor(id string) bool {
	if s, ok := sm.sensors[id]; ok { // o ok serve pra verificar se o sensor com o ID fornecido existe no mapa de sensores
		s.Active = !s.Active // Alterna o estado ativo do sensor
		return true // Retorna true se o sensor foi encontrado e seu estado foi alternado com sucesso
	}
	return false // Retorna false se o sensor com o ID fornecido não foi encontrado

}


func (sm *SensorManager) RemoveSensor(id string) {
	delete(sm.sensors, id) // Remove o sensor do mapa de sensores usando a função delete
}

//Simula leituras baseadas no e posição 
func (sm *SensorManager) GenerateReading(sensor *models.Sensor, climate models.ClimateData) models.SensorReading {
	reading := models.SensorReading{
		SensorID: sensor.ID, // ID do sensor que coletou a leitura
		Type: sensor.Type, // Tipo do sensor (e.g., "temperature", "humidity")
		Position: sensor.Position, // Posição do sensor
		Climatete: climate, // Dados climáticos associados à leitura
		Timestamp: time.Now(), // Timestamp da leitura
	}

	// altura afeta temperatura (gradiente de -0.65 °C por 100 metros)
	altitudeEffect := sensor.Position.Y * -0.0065 // Converte a altura para o efeito de temperatura

	switch sensor.Type {
	case "temperature":
		reading.Value = climate.Temperature + altitudeEffect + rand.Float64()*2.0 - 1.0 // Adiciona variação aleatória de ±1°C
		reading.Unit = "°C"

		case "humidity":
			// := significa que a variável é declarada e inicializada ao mesmo tempo,
			//  ou seja, é uma forma de criar uma nova variável e atribuir um valor 
			// a ela em uma única linha de código. No caso do heightFactor, ele é 
			// calculado com base na posição Y do sensor, onde sensores mais altos 
			// terão um fator menor, refletindo a diminuição da umidade relativa com a altura.
			// umidade relativa varia com altura (geralmente menor em cima)
			heighFactor := 1.0 - (sensor.Position.Y / 1000.0) // Fator de redução de umidade com a altura
			reading.Value = climate.Humidity * heightFactor + rand.Float64()*5.0 - 2.5 // Adiciona variação aleatória de ±2.5%
			reading.Unit = "%"

			// um exemplo de uso é para um sensor de umidade
			//  localizado a 500 metros de altura, o heightFactor 
			// seria calculado como 1.0 - (500 / 1000) = 0.5, 
			// o que significa que a umidade relativa medida por esse 
			// sensor seria reduzida pela metade em comparação com um sensor ao nível do mar,
			//  refletindo a tendência de diminuição da umidade com a altitude.
			
		case "rain":
			// pluviômetro - acumula a chuva 
			reading.Value = climate.RainIntensity 
			reading.Unit = "mm/hr" // A unidade de medida para a intensidade da chuva é milímetros por hora (mm/hr), que indica a quantidade de chuva que cai em um determinado período de tempo. Por exemplo, uma leitura de 10 mm/hr significa que 10 milímetros de chuva estão caindo em uma hora.

		case "soil_moisture":
			// umidade do solo - afeta a umidade relativa e a temperatura (evaporação)
			baseMoisture := 30.0 // Umidade base do solo em porcentagem

			// chuva aumenta umidade com delay de absorção
			rainEffect := climate.RainIntensity * 0.5 // A chuva aumenta a umidade do solo, mas com um fator de absorção
			 
			// profundidade: mais úmido embaixo (assintota) 
			depthFactor := 1.0 - math.Exp(-sensor.Position.Z/100.0) // Fator de aumento de umidade com a profundidade do solo

			//Evaporação pelo sol 
			evaporation := climate.SolarRadiation / 1000.0 * 5.0 

			reading.Value = (baseMoisture + rainEffect) * depthFactor - evaporation + rand.Float64()*5.0 - 2.5 // Adiciona variação aleatória de ±2.5%
			if reading.Value > 100 {
				reading.Value = 100.0 // Limita a umidade do solo a 100%
			} 
			if reading.Value < 0 {
				reading.Value = 0.0 // Limita a umidade do solo a 0%
			}
			reading.Unit = "%"


		case "wind":
			//velocidade do vento aumenta com altura (lei da potência diz que o vento aumenta com a altura, mas de forma não linear)
			heighFactor := math.Pow(1.0+sensor.Position.Y/100.0, 0.2) // Fator de aumento de vento com a altura
			reading.Value = climate.WindSpeed * heightFactor + rand.Float64()*3.0 - 1.5 // Adiciona variação aleatória de ±1.5 m/s
			reading.Unit = "km/h"

		case "solar":
			// radiação solar baseada na hora do dia e posição (sombra)
			reading.Value = climate.SolarRadiation * (1.0 - sensor.Position.Y/100.0) 
			if reading.Value < 0 {
				reading.Value = 0.0 // Limita a radiação solar a 0
			}
			reading.Unit = "W/m²"
	}

	return reading // Retorna a leitura gerada para o sensor
}

func (sm *SensorManager) Start() {
	climateCh := sm.climateSim.Subscribe() // Inscreve-se para receber atualizações climáticas do simulador de clima

	go func() {
		ticker := time.NewTicker(1 * time.Second) // Configura um ticker para gerar leituras a cada segundo
		for {
			climate := <-climateCh

			for _, sensor := range sm.sensors {
				if !sensor.Active {
					continue // Pula sensores inativos
				}

				reading := sm.GenerateReading(sensor, climate) // Gera uma leitura para o sensor com base nos dados climáticos atuais

				//Publica no MQTT 
				topic := "agro/sensors/" + sensor.ID.String() // Define o tópico MQTT usando o ID do sensor
				payload, _ := json.Marshal(reading) // Converte a leitura para JSON
				sm.mqttPublish(topic, payload) // Publica a leitura no tópico MQTT

				select {
				case sm.readings <- reading: 
				default:
               }
            }
        }
    }()
}

