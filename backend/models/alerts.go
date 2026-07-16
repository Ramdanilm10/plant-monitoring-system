package models

type Alert struct {

	PlantID int `json:"plant_id"`
	Status string `json:"status"`
	Message string `json:"message"`
	Recommendation string `json:"recommendation"`

}