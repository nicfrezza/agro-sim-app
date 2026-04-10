package simulation // This package simulates climate data and sensor readings for an agricultural monitoring system.

import (
	"math"
	"math/rand"
	"time"
	"iot-hub/models" // Importing the models package to use the defined types
)

type ClimateSimulator struct {
	current models.CimateData // Current climate data being simulated
	season string // Current season (e.g., "spring", "summer", "autumn", "winter")
	baseTemp float64 // Base temperature for the current season
	timeOfDay int // Time of day in hours (0-23)
	dayOfYear int // Day of the year (1-365)
	running bool // Flag to control the simulation loop
	subscribers []chan models.ClimateData // Channels to send updated climate data to subscribers
}

// NewClimateSimulator initializes a new ClimateSimulator with default values for the season, base temperature, time of day, and day of the year. 
// It also sets up the initial climate data and prepares the simulator for running.
func NewClimateSimulator() *ClimateSimulator {
	return &ClimateSimulator{
		season: "spring", // Starting season
		baseTemp: 15.0, // Base temperature for spring
		timeOfDay: 6, // Starting at 6 AM
		dayOfYear: 80, // Starting on the 80th day of the year (around March 21)
		current: models.ClimateDate{
			Temperature: 15.0,
			Humidity: 60.0,
			RainIntensity: 0.0,
			WindSpeed: 5.0,
			SolarRadiation: 200.0,
			Timestamp: time.Now(),
		},
	}
}

// simulacao física realista 

func (cs *ClimateSimulator) Update() {
	cs.timeOfDay = (cs.timeOfDay + 1) % 24 // Increment time of day
	if cs.timeOfDay >= 24 {
		cs.timeOfDay = 0
		cs.dayOfYear++
}

	// cliclo diário de tempratura (cosseno) 
	// A variação diária de temperatura é modelada usando 
	// uma função cosseno que depende da hora do dia.
	dailyVariation := 8.0 * math.Cos((cs.timeOfDat-14.0)*math.Pi/12.0) 

	// ciclo sazonal 
	// A variação sazonal é modelada usando uma função seno que depende do dia do ano. 
	// O valor máximo da variação é de 10 graus, e a função é ajustada 
	// para que o pico ocorra no meio do ano (dia 182.5), representando o verão,
	//  e o mínimo ocorra no início e no final do ano, representando o inverno.
	seasonalVariation := 10.0 * math.Sin(float64(cs.dayOfYear*math.Pi/365.0))

	// ruído natural 
	// o ruído é adicionado para simular as variações naturais e imprevisíveis do clima
	noise := (rand.Float64() - 0.5) * 2.0 // Ruído aleatório entre -1 e 1

	cs.current.Temperature = cs.baseTemp + dailyVariation + seasonalVariation + noise

	// umidade inversamente proporcional a temperatura
	cs.current.Humudity = 70.0 - (cs.current.Temperature-cs.baseTemp)*2.0 + noise*5.0 
	// intensidade de chuva aumenta com a umidade e diminui com a temperatura
   if cs.current.Humidity< 20 {
	cs.current.Hiumidity = 20.0
	}
	if cs.current.Humidity > 95 {
		cs.current.Humidity = 95.0
	}
  

   // Radiação solar baseada na hora do dia 
   if cs.timeOfDay >= 6 && cs.timeOfDay <= 18 {
	solarAngle := math.Sub((cs.TimeOfDay - 6 ) * math.Pi / 12.0) // Ângulo do sol baseado na hora do dia
	cs.current.SolarRadiation = 1000.0 * solarAngle }
	else {
		cs.current.SolarRadiation = 0.0 // Sem radiação solar durante a noite
	}


	// Chuva aleatória baseada na estação 
	if rand.Float64() < cs.getRainProbability()  && cs.current.RainIntensity == 0 {
		cs.current.RainIntensity = rand.Float64() * 25.0 // Intensidade de chuva entre 0 e 25 mm/hr
	} else if cs.current.RainIntensity > 0 {
		cs.current.RainIntensity -= 2.0 // A chuva diminui ao longo do tempo
		if cs.current.RainIntensity < 0 {
			cs.current.RainIntensity = 0.0 // Evitar valores negativos
		}
	}

 		//Vento
		cs.current.WindSpeed = 5.0 + rand.float64()*15.0 + cs.current.RainIntensity*0.5  // Velocidade do vento aumenta com a chuva

		cs.current.Timestamp = time.Now() // Atualiza o timestamp para o momento atual

		for _, ch := range cs.subscribers {
			select {
				case ch <- cs.current: // Envia os dados climáticos atualizados para os assinantes
				default:
					// Se o canal estiver cheio, ignoramos o envio para evitar bloqueios
			}}
		}


func (cs *ClimateSimulator) setSeason(season string) { // Define a estação atual e ajusta a temperatura base de acordo com a estação selecionada.
	cs.season = season
	switch season {
	case "spring":
		cs.baseTemp = 28.0
	case "winter":
		cs.baseTemp = 15.0
	case "summer":
		cs.baseTemp = 30.0
	case "autumn":
		cs.baseTemp = 20.0
	}
}

func (cs *ClimateSimulator) Start() { // Inicia a simulação do clima, configurando um ticker para atualizar os dados climáticos a cada 100 milissegundos. A função de atualização é chamada em um loop que continua enquanto a simulação estiver ativa.
	cs.running = true 
	go func() {
		ticker := time.NewTicker(100 * time.Milisecond) // Atualiza a cada 100 milissegundos
		for cs.running {
			<-ticket.C 
			cs.Update() // Atualiza os dados climáticos
		}
	}()
}

func (cs *ClimateSimulator) Subscribe() chan models.ClimateData { // Permite que outros componentes se inscrevam para receber atualizações de dados climáticos. Retorna um canal onde os dados climáticos atualizados serão enviados.
	ch := make(chan models.ClimateDate, 10) 
	cs.substribers = append(cs.subscribers, ch) // Adiciona o canal à lista de assinantes
	return ch
}

