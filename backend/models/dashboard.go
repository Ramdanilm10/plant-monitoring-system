package models

type DashboardData struct {

	Plant Plant
	Sensor SensorLog
	Condition Alert

}

type Plant struct {

	ID int `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`

}