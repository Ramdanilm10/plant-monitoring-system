package controllers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"plant-monitoring-backend/services"
)

func TestBlynk(c *gin.Context){

	value, err := services.GetBlynkValue("V0")

	if err != nil {

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error":err.Error(),
			},
		)

		return
	}


	c.JSON(
		http.StatusOK,
		gin.H{
			"soil":value,
		},
	)
}