package controller

import (
	"encoding/json"

	"net/http"

	"fmt"

	"net"

	"github.com/qianlifu12345/flat-networking-ipam/rest-service/model"

	"github.com/qianlifu12345/flat-networking-ipam/rest-service/store"
)

//IPController implement BaseController
type IPController struct {
	Controller
}

// IPReq request model
type IPReq struct {
	RequestedIP net.IP      `json:"requestedIp"`
	Subnet      model.IPNet `json:"subnet"`
}

//Post post method
func (c *IPController) Post() {
	var (
		req   IPReq
		err   error
		newIP net.IP
	)
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Error(http.StatusNotAcceptable, fmt.Sprintf("request error:%s", err))
		return
	}
	if !subnetMap.Has(req.Subnet.String()) {
		c.Error(http.StatusNotAcceptable, "The subnet specified hasn't init, you may request POST /subnet first")
		return
	}

	tmp, _ := subnetMap.Get(req.Subnet.String())
	subnet := tmp.(*(model.Subnetwork))
	ips := &(subnet.Ips)

	if req.RequestedIP != nil {
		if model.Contains(&(req.RequestedIP), ips) {
			c.Error(http.StatusBadRequest, "The requestd IP has been used")
			return
		}
		if !((*net.IPNet)(&(req.Subnet))).Contains(req.RequestedIP) {
			c.Error(http.StatusBadRequest, "The requestd IP isn't within specified subnet")
			return

		}
		newIP = req.RequestedIP
	} else {
		newIP, err = subnet.NextIP()
		if err != nil {
			c.Error(http.StatusInternalServerError, err.Error())
			return
		}
	}

	*ips = append(*ips, newIP)
	err = store.ReserveIP(req.Subnet.String(), newIP.String())
	if err != nil {
		c.Error(http.StatusInternalServerError, "reserver IP error")
		return
	}
	subnet.LastReservedIP = newIP

	c.Data["json"] = &(model.ReservedIP{IP: newIP, Gateway: subnet.Gateway})
	c.ServeJSON()
}

//Get get method
func (c *IPController) Get() {
	var list *[]model.ReservedIP
	for _, k := range subnetMap.Keys() {
		tmp, ok := subnetMap.Get(k)
		if !ok {
			c.Error(http.StatusInternalServerError, "internal error")
		}
		subnet, ok := tmp.(*model.Subnetwork)
		if !ok {
			c.Error(http.StatusInternalServerError, "internal error")
		}
		ips := &(subnet.Ips)

		for _, ip := range *ips {
			if list == nil {
				*list = []model.ReservedIP{model.ReservedIP{IP: ip, Gateway: subnet.Gateway}}
			} else {
				*list = append(*list, model.ReservedIP{IP: ip, Gateway: subnet.Gateway})
			}
		}
	}
	v := ResponseIP{Data: *list, Type: "collection"}
	c.Data["json"] = &v
	c.ServeJSON()
}

type ResponseIP struct {
	Type string             `json:"type"`
	Data []model.ReservedIP `json:"data"`
}

//Delete delete method
func (c *IPController) Delete() {
	var req = c.GetString("ip")
	if req == "" {
		c.Error(http.StatusBadRequest, "no ip specified")
		return
	}
	ip, ipNet, err := net.ParseCIDR(req)
	if err != nil {
		c.Error(http.StatusBadRequest, err.Error())
		return
	}

	tmp, _ := subnetMap.Get(ipNet.String())
	subnet := tmp.(*(model.Subnetwork))
	ips := &(subnet.Ips)

	for i, v := range *ips {
		if v.Equal(ip) {
			if subnet.LastReservedIP.Equal(ip) {
				subnet.LastReservedIP, _ = subnet.NextIP()
			}
			*ips = append((*ips)[:i], (*ips)[i+1:]...)
			err = store.Store(subnet)
			if err != nil {
				c.Error(http.StatusInternalServerError, err.Error())
				return
			}
		}
	}
}
