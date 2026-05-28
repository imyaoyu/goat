package main

import (
	"github.com/imyaoyu/goat/app"
)

func main() {

	//Init sql
	if sql := app.Env("SQL"); sql != "" {
		if _, err := app.DB().Exec(sql); err != nil {
			panic(err)
		}
	}

	app.CodeMsg(500, "SystemError")

	//Define api
	app.Api("1", SignUpOrIn)

	//Define middleware
	app.Func(LogIp)

	//Start Server
	app.Run()

}

// middlewaire to log forward ip
func LogIp(c *app.ApiCtx) {

	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		c.Log("LogIP", "X-Forwarded-For", xff)
	}

}

// input
type iSignUpOrIn struct {
	OpenId string `validate:"min=10,max=50"`
}

// output
type oSignUpOrIn struct {
	UserId     string
	LoginCount int
}

// table t_app_user
type tAppUser struct {
	Id          int64
	UserId      string
	OpenId      string
	LoginCount  int
	CreatedTime string
	UpdatedTime string
}

func SignUpOrIn(c *app.ApiCtx) {
	i, o := new(iSignUpOrIn), new(oSignUpOrIn)
	c.Init(i, o)
	user := new(tAppUser)
	if has := c.Select(user, "open_id=?", i.OpenId); has {
		user.LoginCount++
		c.Update(user, user.Id, map[string]any{
			"login_count":  user.LoginCount,
			"updated_time": app.Now(),
		})
	} else {
		user.OpenId = i.OpenId
		user.UserId = app.UUID()
		user.LoginCount = 1
		user.CreatedTime = app.Now()
		user.UpdatedTime = user.CreatedTime
		c.Insert(user)
	}
	o.UserId = user.UserId
	o.LoginCount = user.LoginCount

}
