package main

import (
	"github.com/imyaoyu/goat/api"
	"github.com/imyaoyu/goat/app"
)

func main() {

	//Init sql (optional), you can use other method to init database
	if sql := app.Env("SQL"); sql != "" {
		if _, err := app.DB().Exec(sql); err != nil {
			panic(err)
		}
	}

	//Define error message(optional), it can be repalce the default respoonse message
	app.CodeMsg(500, "SystemError")

	//Define middleware(optional)
	app.Func(LogIp)

	//Define api
	app.Api("a.1", api.SignUpOrIn)

	app.Api("demo", func(c *app.ApiCtx) {

		type Ping struct {
			Msg string
		}
		type Pong struct {
			Msg string
		}

		i, o := new(Ping), new(Pong)
		c.Init(i, o)

		o.Msg = i.Msg
		//End

	})

	//Start Server
	app.Run()

}

// middlewaire to log forward ip
func LogIp(c *app.ApiCtx) {

	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		c.Log("LogIP", "X-Forwarded-For", xff)
	}

}
