package zoomlion

import (
	"encoding/json"
	"fmt"
	"github.com/rancher/rancher/pkg/api/store/auth"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
	"github.com/rancher/rancher/pkg/auth/providers/common"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/apis/management.cattle.io/v3public"
	"github.com/rancher/types/client/management/v3"
)

func (g *zlProvider) formatter(apiContext *types.APIContext, resource *types.RawResource) {
	common.AddCommonActions(apiContext, resource)
	resource.AddAction(apiContext, "configureTest")
	resource.AddAction(apiContext, "testAndApply")
}

func (g *zlProvider) actionHandler(actionName string, action *types.Action, request *types.APIContext) error {
	handled, err := common.HandleCommonAction(actionName, action, request, Name, g.authConfigs)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	if actionName == "configureTest" {
		return g.configureTest(actionName, action, request)
	} else if actionName == "testAndApply" {
		return g.testAndApply(actionName, action, request)
	}

	return httperror.NewAPIError(httperror.ActionNotAvailable, "")
}

func (g *zlProvider) configureTest(actionName string, action *types.Action, request *types.APIContext) error {
	zoomlionConfig := &v3.ZoomlionConfig{}
	if err := json.NewDecoder(request.Request.Body).Decode(zoomlionConfig); err != nil {
		return httperror.NewAPIError(httperror.InvalidBodyContent,
			fmt.Sprintf("Failed to parse body: %v", err))
	}
	redirectURL := formZoomlionRedirectURL(zoomlionConfig)

	data := map[string]interface{}{
		"redirectUrl": redirectURL,
		"type":        "zoomlionConfigTestOutput",
	}

	request.WriteResponse(http.StatusOK, data)
	return nil
}

func formZoomlionRedirectURL(zlConfig *v3.ZoomlionConfig) string {
	return zoomlionRedirectURL(zlConfig.Hostname, zlConfig.ClientID, zlConfig.TLS)
}

func formZoomlionRedirectURLFromMap(config map[string]interface{}) string {
	hostname, _ := config[client.ZoomlionConfigFieldHostname].(string)
	clientID, _ := config[client.ZoomlionConfigFieldClientID].(string)
	tls, _ := config[client.ZoomlionConfigFieldTLS].(bool)
	return zoomlionRedirectURL(hostname, clientID, tls)
}

func zoomlionRedirectURL(hostname, clientID string, tls bool) string {
	redirect := ""
	if hostname != "" {
		scheme := "http://"
		if tls {
			scheme = "https://"
		}
		redirect = scheme + hostname
	} else {
		redirect = zoomlionDefaultHostName
	}
	redirect = redirect + "/oauth2/auth?client_id=" + clientID
	return redirect
}

func (g *zlProvider) testAndApply(actionName string, action *types.Action, request *types.APIContext) error {
	var zoomlionConfig v3.ZoomlionConfig
	zoomlionConfigApplyInput := &v3.ZoomlionConfigApplyInput{}

	if err := json.NewDecoder(request.Request.Body).Decode(zoomlionConfigApplyInput); err != nil {
		return httperror.NewAPIError(httperror.InvalidBodyContent,
			fmt.Sprintf("Failed to parse body: %v", err))
	}
	zoomlionConfig = zoomlionConfigApplyInput.ZoomlionConfig
	zoomlionLogin := &v3public.ZoomlionLogin{
		Code: zoomlionConfigApplyInput.Code,
	}

	if zoomlionConfig.ClientSecret != "" {
		value, err := common.ReadFromSecret(g.secrets, zoomlionConfig.ClientSecret,
			strings.ToLower(auth.TypeToField[client.ZoomlionConfigType]))
		if err != nil {
			return err
		}
		zoomlionConfig.ClientSecret = value
	}

	//Call provider to testLogin
	userPrincipal, groupPrincipals, providerInfo, err := g.LoginUser(zoomlionLogin, &zoomlionConfig, true)
	if err != nil {
		if httperror.IsAPIError(err) {
			return err
		}
		return errors.Wrap(err, "server error while authenticating")
	}

	//if this works, save zoomlionConfig CR adding enabled flag
	user, err := g.userMGR.SetPrincipalOnCurrentUser(request, userPrincipal)
	if err != nil {
		return err
	}

	zoomlionConfig.Enabled = zoomlionConfigApplyInput.Enabled
	err = g.saveZoomlionConfig(&zoomlionConfig)
	if err != nil {
		return httperror.NewAPIError(httperror.ServerError, fmt.Sprintf("Failed to save zoomlion config: %v", err))
	}

	return g.tokenMGR.CreateTokenAndSetCookie(user.Name, userPrincipal, groupPrincipals, providerInfo, 0, "Token via zoomlion Configuration", request)
}
