package models 

import {
	"time"
	"github.com/google/uuid" // Importing the uuid package for generating unique identifiers
}

type Position struct {
	X, Y, Z float64 `json:"x, y,z"` // Coordinates in 3D space
}

type Sensor struct {
	ID        uuid.UUID `json:"id"` // Unique identifier for the sensor
	Type      string    `json:"type"` // "temperature", "humidity", "rain", "soil_moisture"
	Position Position `json:"position"`
    Active   bool     `json:"active"`
    Name     string   `json:"name"`
}

type ClimateData struct {
	Temperature   float64   `json:"temperature"` // in Celsius
	Humidity      float64   `json:"humidity"`    // in percentage
	RainIntensity float64   `json:"rain_intensity"` // in mm/hr
	WindSpeed     float64   `json:"wind_speed"` // in m/s
	SolarRadiation float64   `json:"solar_radiation"` // in W/m²
	Timestamp     time.Time `json:"timestamp"` // Time of data collection
}


type SensorReading struct {
	SensorID    uuid.UUID   `json:"sensor_id"` // ID of the sensor that collected the data
	Type		string      `json:"type"` // "temperature", "humidity", "rain", "soil_moisture"
	Value       float64     `json:"value"` // The actual reading value
	Unit		string      `json:"unit"` // Unit of the reading (e.g., "C", "%", "mm/hr")
	Position    Position    `json:"position"` // Position of the sensor when the reading was taken
	Climatete  ClimateData `json:"climate_data"` // Associated climate data at the time of reading	
	Timestamp   time.Time   `json:"timestamp"` // Time when the reading was taken
}

func NewSensor(sensorType string, position Position, name string) *Sensor {
	return &Sensor{
		ID:       uuid.New(), // Generate a new unique identifier for the sensor
		Type:     sensorType,
		Position: position,
		Active:   true, // Sensors are active by default when created
		Name:     name,
	}
}

