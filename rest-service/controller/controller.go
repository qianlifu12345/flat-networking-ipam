package controller

import "github.com/astaxie/beego"

type errorResponse struct {
	Success bool `json:"success"`
	Message  string `json:"message"`
}

// Controller define
type Controller struct {
	beego.Controller
}

// Abort common error handler
func (c *Controller) Error(status int,msg string) {
	err := errorResponse{false, msg}
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = err
	c.ServeJSON()
}
