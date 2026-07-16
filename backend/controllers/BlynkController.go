package controllers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"plant-monitoring-backend/services"
)

func GetLiveSensor(c *gin.Context) {

	soil1, err := services.GetBlynkValue("V0")

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	soil2, _ := services.GetBlynkValue("V1")

	temperature, _ := services.GetBlynkValue("V2")

	humidity, _ := services.GetBlynkValue("V3")

	c.JSON(
		http.StatusOK,
		gin.H{

			"soil_1": soil1,

			"soil_2": soil2,

			"temperature": temperature,

			"humidity": humidity,
		},
	)

}
