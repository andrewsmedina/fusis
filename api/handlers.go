package api

import (
	"fmt"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/luizbafilho/fusis/ipvs"
)

func (as ApiService) serviceList(c *gin.Context) {
	services := as.balancer.GetServices()
	if len(services) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, services)
}

func (as ApiService) serviceGet(c *gin.Context) {
	serviceId := c.Param("service_id")
	service, err := as.balancer.GetService(serviceId)
	if err != nil {
		if err == ipvs.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("GetService() failed: %v", err)})
		}
		return
	}
	c.JSON(http.StatusOK, service)
}

func (as ApiService) serviceCreate(c *gin.Context) {
	var newService ipvs.Service
	if err := c.BindJSON(&newService); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//Guarantees that no one tries to create a destination together with a service
	newService.Destinations = []ipvs.Destination{}

	if _, errs := govalidator.ValidateStruct(newService); errs != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": govalidator.ErrorsByField(errs)})
		return
	}

	if _, err := newService.ValidateUniqueness(); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// If everthing is ok send it to Raft
	err := as.balancer.AddService(&newService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("UpsertService() failed: %v", err)})
		return
	}

	c.Header("Location", fmt.Sprintf("/services/%s", newService.Id))
	c.JSON(http.StatusCreated, newService)
}

func (as ApiService) serviceDelete(c *gin.Context) {
	serviceId := c.Param("service_id")
	_, err := as.balancer.GetService(serviceId)
	if err != nil {
		if err == ipvs.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("GetService() failed: %v", err)})
		}
		return
	}

	err = as.balancer.DeleteService(serviceId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("DeleteService() failed: %v\n", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

func (as ApiService) destinationCreate(c *gin.Context) {
	serviceId := c.Param("service_id")
	service, err := as.balancer.GetService(serviceId)
	if err != nil {
		if err == ipvs.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("GetService() failed: %v", err)})
		}
		return
	}

	destination := &ipvs.Destination{Weight: 1, Mode: "route", ServiceId: serviceId}
	if err := c.BindJSON(destination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, errs := govalidator.ValidateStruct(destination); errs != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": govalidator.ErrorsByField(errs)})
		return
	}

	if _, err := destination.ValidateUniqueness(service); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	err = as.balancer.AddDestination(service, destination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("UpsertDestination() failed: %v\n", err)})
		return
	}

	c.Header("Location", fmt.Sprintf("/services/%s/destinations/%s", serviceId, destination.Id))
	c.JSON(http.StatusCreated, destination)
}

func (as ApiService) destinationDelete(c *gin.Context) {
	destinationId := c.Param("destination_id")
	dst, err := as.balancer.GetDestination(destinationId)
	if err != nil {
		if err == ipvs.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprint("Destination not found")})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("GetDestination() failed: %v", err)})
		}
		return
	}

	err = as.balancer.DeleteDestination(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("DeleteDestination() failed: %v\n", err)})
	}

	c.Data(http.StatusOK, gin.MIMEHTML, nil)
}

func (as ApiService) flush(c *gin.Context) {
	// err := as.ipvs.Flush()
	// if err != nil {
	// 	c.JSON(400, gin.H{"error": err.Error()})
	// 	return
	// }
	//
	// err = ipvs.Flush()
	// if err != nil {
	// 	c.JSON(400, gin.H{"error": err.Error()})
	// 	return
	// }
}
