package services

import (

"plant-monitoring-backend/config"
"plant-monitoring-backend/models"

)

func GetPlantRule(plantID int) (models.PlantRule,error){

var rule models.PlantRule
query := `

SELECT

plant_id,
min_temperature,
max_temperature,
min_humidity,
max_humidity,
min_soil_moisture,
max_soil_moisture

FROM plant_rules
WHERE plant_id=$1

`

err := config.DB.QueryRow(
	query,
	plantID,
).Scan(

&rule.PlantID,
&rule.MinTemperature,
&rule.MaxTemperature,
&rule.MinHumidity,
&rule.MaxHumidity,
&rule.MinSoilMoisture,
&rule.MaxSoilMoisture,

)

return rule,err

}