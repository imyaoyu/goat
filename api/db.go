package api

// table t_app_user
type tAppUser struct {
	Id          int64
	UserId      string
	OpenId      string
	LoginCount  int
	CreatedTime string
	UpdatedTime string
}
