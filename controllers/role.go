package controllers

import (
	"HealthHub360/config/authorization"
	"HealthHub360/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Role(router *gin.Engine) {

	router.POST("/role/create", CreateRole)
	role := router.Group("/role", authorization.JWTAuth())
	{
		role.POST("/fetchAll", authorization.Authorize("role", "view"), ReadRoles)
		role.POST("/update/:roleCode", authorization.Authorize("role", "update"), UpdateRole)
		role.GET("/fetch/:roleCode", authorization.Authorize("role", "view"), FetchRoleById)
		role.DELETE("/delete/:roleCode", authorization.Authorize("role", "delete"), DeleteRole)
	}
}

/*
* Take the json format
* Pass to prepareData
* Move to services with the parameter context and map[string]interface
 */
func CreateRole(c *gin.Context) {

	var roleData map[string]interface{}

	if err := c.BindJSON(&roleData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	insertedRole, err := services.CreateRole(c, roleData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role created successfully",
		"data":    insertedRole,
	})
}

/*
Here It reads all the roles of the user
*/
func ReadRoles(c *gin.Context) {
	data, err := services.ReadRoles(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"Roles": data,
	})
}

/*
Here it updates the role of the role by taking the code from param
*/
func UpdateRole(c *gin.Context) {
	roleCode := c.Param("roleCode")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	updated, err := services.UpdateRole(c, roleCode, body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, updated)
}

/*
Fetch role by its id using rolecode given in param
*/
func FetchRoleById(c *gin.Context) {
	roleCode := c.Param("roleCode")
	var body map[string]interface{}
	body, err := services.FetchRoleById(c, roleCode)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"Data": body})
}

/*
Here it deletes the role of bty taking the roleid as a
param and perform the delete operation
*/
func DeleteRole(c *gin.Context) {
	log.Println("Adi")
	roleCode := c.Param("roleCode")
	err := services.DeleteRole(c, roleCode)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "success"})
}
