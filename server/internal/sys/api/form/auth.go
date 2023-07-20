package form

type OAuth2Form struct {
	ClientID         string `json:"clientID" binding:"required"`
	ClientSecret     string `json:"clientSecret" binding:"required"`
	AuthorizationURL string `json:"authorizationURL" binding:"required,url"`
	AccessTokenURL   string `json:"accessTokenURL" binding:"required,url"`
	ResourceURL      string `json:"resourceURL" binding:"required,url"`
	RedirectURL      string `json:"redirectURL" binding:"required,url"`
	UserIdentifier   string `json:"userIdentifier" binding:"required"`
	Scopes           string `json:"scopes"`
}
