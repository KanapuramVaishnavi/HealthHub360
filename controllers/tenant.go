package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Tenant(router *gin.Engine) {
	tenant := router.Group("/tenant", authorization.JWTAuth())
	{
		tenant.POST("/create", authorization.Authorize("tenant", "create"), CreateTenant)
		tenant.GET("/fetchAll", authorization.Authorize("tenant", "view"), FetchAll)
		tenant.POST("/update/:code", authorization.Authorize("tenant", "update"), UpdateTenant)
		tenant.DELETE("/delete/:code", authorization.Authorize("Tenant", "delete"), DeleteTenantByCode)
	}
}

func CreateTenant(c *gin.Context) {
	tenant := make(map[string]interface{})
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.CreateTenant(c, tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tenant registered successfully!"})
}

/*
Here the fetching of All Tenants will Happen and returns
Error if it had any respectively
and move into services
*/
func FetchAll(c *gin.Context) {
	results, err := services.FetchAllTenants(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenants": results})
}

/*
Here the Updation of Tenant will Happen it takes the
map of data and binds it to it respectively
and move into services
*/
func UpdateTenant(c *gin.Context) {
	code := c.Param("code")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	updated, err := services.UpdateTenantByCode(c, code, body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	log.Println(updated)
	c.JSON(200, gin.H{
		"Success": "Updated Successfully",
	})
}

/*
Here the Deletion of Tenant will Happen it takes the
Code and checks wthether the code is there or not respectively
and move into services
*/
func DeleteTenantByCode(c *gin.Context) {
	code := c.Param("code")
	err := services.DeleteTenantByCode(c, code)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"Success": "Updated Successfully",
	})
}
