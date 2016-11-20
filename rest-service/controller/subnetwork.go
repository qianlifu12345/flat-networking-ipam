package controller

import (
	"encoding/json"

	"net/http"

	"fmt"

	"ipam-htsc/rest-service/model"
	"ipam-htsc/rest-service/store"
)

//SubnetworkController implement BaseController
type SubnetworkController struct {
	Controller
}

var subnetMap = store.LoadIPAMConfig()

//Post post method
// @router /subnet [post]
func (c *SubnetworkController) Post() {
	var req model.Subnetwork

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Error(http.StatusNotAcceptable, fmt.Sprintf("request error:%s", err))
		return
	}
	if subnetMap.Has(req.Subnet.String()) {
		return
	}
	subnetMap.Set(req.Subnet.String(), &req)
	if err := store.NewIPNet(req.Subnet.String(), req.String()); err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
}

//Get get method
// @router /subnet/ [get]
func (c *SubnetworkController) Get() {
	c.Data["json"] = &subnetMap
	c.ServeJSON()
}

//Delete delete method
// @router /subnet [delete]
func (c *SubnetworkController) Delete() {
	var subnet = c.GetString("subnet")
	subnetMap.Remove(subnet)
}