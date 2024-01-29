package api

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VladimirMarkelov/clui"
	"github.com/valyala/fastjson"
	"main.go/src/requests"
	"main.go/src/utils"
)

func fetch(url string, method string, args ...interface{}) (http.Header, *fastjson.Value, error) {
	req := requests.Requests()
	req.SetTimeout(3)
	req.Client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	actions := map[string](func(url string, args ...interface{}) (resp *requests.Response, err error)) {
		"Get": req.Get,
		"Post": req.Post,
		"PostJson": req.PostJson,
	}
	resp, err := actions[method](url, args...)
	if err != nil {
		return nil, nil, err
	}
	if resp.R.StatusCode >= 400 {
		return resp.R.Header, nil, fmt.Errorf("response status: %d", resp.R.StatusCode)
	}

	var p fastjson.Parser
	data, err := p.Parse(resp.Text())
	if err != nil {
		data = nil
	}
	return resp.R.Header, data, nil
}

func Login(host *fastjson.Value, tokens *fastjson.Value) (string, error) {
	hostname := utils.GetConfigFieldValue(host, "HostName")
	token := utils.GetConfigFieldValue(tokens, hostname, "token")
	timestap := tokens.GetInt64(hostname, "time")
	now := time.Now().Unix()
	if token != "" && (now - timestap) < 1200 {
		return token, nil
	}

	ip := utils.GetConfigFieldValue(host, "ControllerIP")
	username := utils.GetConfigFieldValue(host, "username")
	password := utils.GetConfigFieldValue(host, "password")

	header, body, err := fetch("https://" + ip  + "/redfish/v1", "Get")
	if err != nil { return "", err }

	version, err := strconv.Atoi(strings.Replace(utils.GetConfigFieldValue(body, "RedfishVersion"), ".", "", -1))
	session_uri := ""
	if version >= 160 {
		session_uri = "/redfish/v1/SessionService/Sessions"
	} else if version < 160 {
		session_uri = "/redfish/v1/Sessions"
	} else {
		return "", errors.New("version error")
	}
	url := "https://" + ip + session_uri
	payload := fmt.Sprintf(`{"UserName":"%s","Password":"%s"}`, username, password)
	header, body, err = fetch(url, "PostJson", payload)
	if err != nil { return "", err }

	token = header.Get("X-Auth-Token")
	var a fastjson.Arena
	tokens.Get(hostname).Set("token", a.NewString(token))
	tokens.Get(hostname).Set("time", a.NewNumberFloat64(float64(time.Now().Unix())))
	utils.SaveToken("", tokens)
	return token, nil
}

func setPowerState(host *fastjson.Value, tokens *fastjson.Value, state string) (string, error) {
	token, err := Login(host, tokens)
	if err != nil { return "", err }

	ip := utils.GetConfigFieldValue(host, "ControllerIP")
	url := "https://" + ip + "/redfish/v1/Systems/System.Embedded.1/Actions/ComputerSystem.Reset"
	payload := fmt.Sprintf(`{"ResetType": "%s}`, state)
	_, body, err := fetch(url, "PostJson", payload, requests.Header{"X-Auth-Token": token})
	if err != nil { return "", err }

	return utils.GetConfigFieldValue(body, "PowerState"), nil
}

func PowerOffHost(host *fastjson.Value, tokens *fastjson.Value, force bool) (string, error) {
	var state string
	if force {
		state = "ForceOff"
	} else {
		state = "GracefulShutdown"
	}
	return setPowerState(host, tokens, state)
}
func PowerOnHost(host *fastjson.Value, tokens *fastjson.Value) (string, error) {
	return setPowerState(host, tokens, "On")
}
func GetPowerState(host *fastjson.Value, tokens *fastjson.Value) (state string, err error) {
	defer func () {
		var a fastjson.Arena
		if err != nil {
			host.Set("Network Stat", a.NewString("Error"))
			host.Set("Power Stat", a.NewString("Error"))
		} else {
			host.Set("Network Stat", a.NewString("Online"))
			host.Set("Power Stat", a.NewString(state))
		}
	}()

	clui.Logger().Printf("get %s power state\n", host.Get("HostName"))

	token, err := Login(host, tokens)
	if err != nil {
		clui.Logger().Printf("login error: %s\n", err.Error())
		return "", err
	}

	ip := utils.GetConfigFieldValue(host, "ControllerIP")
	url := "https://" + ip + "/redfish/v1/Systems/System.Embedded.1"
	_, body, err := fetch(url, "Get", requests.Header{"X-Auth-Token": token})
	if err != nil {
		clui.Logger().Printf("get state error: %s\n", err.Error())
		return "", err
	}

	/**
		允许的状态：
	    "On",
        "ForceOff",
        "GracefulRestart",
        "GracefulShutdown",
        "PushPowerButton",
        "Nmi"
	*/
	return utils.GetConfigFieldValue(body, "PowerState"), nil
}