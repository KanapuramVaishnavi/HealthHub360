package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"HealthHub360/util"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Tenant(router *gin.Engine) {
	tenant := router.Group("/tenant", authorization.JWTAuth())
	{
		tenant.POST("/create", authorization.Authorize("tenant", "create"), CreateTenant)
		tenant.GET("/fetch/:code", authorization.Authorize("tenant", "view"), FetchTenantByCode)
		tenant.GET("/fetchAll", authorization.Authorize("tenant", "view"), FetchAll)
		tenant.PUT("/update/:code", authorization.Authorize("tenant", "update"), UpdateTenant)
		tenant.DELETE("/delete/:code", authorization.Authorize("Tenant", "delete"), DeleteTenantByCode)
	}
}

func CreateTenant(c *gin.Context) {
	tenant := make(map[string]interface{})
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	if err := services.CreateTenant(c, tenant); err != nil {
		c.JSON(http.StatusInternalServerError, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse("Created successfully"))
}

func FetchTenantByCode(c *gin.Context) {
	tenantId := c.Param("code")
	tenant, err := services.FetchTenantByCode(c, tenantId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(tenant))
}

/*
Here the fetching of All Tenants will Happen and returns
Error if it had any respectively
and move into services
*/
func FetchAll(c *gin.Context) {
	results, err := services.FetchAllTenants(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(results))
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
		c.JSON(400, util.FailedResponse(err))
		return
	}

	updated, err := services.UpdateTenantByCode(c, code, body)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	log.Println(updated)
	c.JSON(200, util.SuccessResponse(updated))
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
