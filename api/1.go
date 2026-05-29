package api

import (
	"github.com/imyaoyu/goat/app"
)

// input
type iSignUpOrIn struct {
	OpenId string `validate:"min=10,max=50"`
}

// output
type oSignUpOrIn struct {
	UserId     string
	LoginCount int
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
