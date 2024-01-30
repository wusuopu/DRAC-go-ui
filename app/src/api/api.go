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
	resty "github.com/go-resty/resty/v2"
	"github.com/valyala/fastjson"
	"main.go/src/utils"
)

func fetch(url string, method string, payload string, headers map[string]string) (http.Header, *fastjson.Value, error) {
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	req := client.NewRequest()
	client.SetTimeout(3 * time.Second)
	if headers != nil {
		req.SetHeaders(headers)
	}
	if payload != "" {
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(payload)
	}
	actions := map[string](func(url string) (resp *resty.Response, err error)) {
		"Get": req.Get,
		"Post": req.Post,
	}
	resp, err := actions[method](url)
	if err != nil {
		return nil, nil, err
	}
	 var p fastjson.Parser
	data, err := p.ParseBytes(resp.Body())
	if err != nil {
		data = nil
	}
	if resp.StatusCode() >= 400 {
		return resp.Header(), data, fmt.Errorf("response status: %d", resp.StatusCode())
	}
	return resp.Header(), data, nil
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

	header, body, err := fetch("https://" + ip  + "/redfish/v1", "Get", "", nil)
	if err != nil {
		clui.Logger().Printf("login get version error: %s\n", err.Error())
		return "", err
	}

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
	header, body, err = fetch(url, "Post", payload, nil)
	if err != nil {
		clui.Logger().Printf("login auth error: %s %s %s\n", err.Error(), payload, body.MarshalTo(nil))
		return "", err
	}

	token = header.Get("X-Auth-Token")
	var a fastjson.Arena
	tokens.Get(hostname).Set("token", a.NewString(token))
	tokens.Get(hostname).Set("time", a.NewNumberFloat64(float64(time.Now().Unix())))
	utils.SaveToken("", tokens)
	return token, nil
}

func setPowerState(host *fastjson.Value, tokens *fastjson.Value, state string) (string, error) {
	clui.Logger().Printf("set %s power state %s\n", host.Get("HostName"), state)

	token, err := Login(host, tokens)
	if err != nil { return "", err }

	ip := utils.GetConfigFieldValue(host, "ControllerIP")
	url := "https://" + ip + "/redfish/v1/Systems/System.Embedded.1/Actions/ComputerSystem.Reset"
	payload := fmt.Sprintf(`{"ResetType": "%s}`, state)
	_, body, err := fetch(url, "Post", payload, map[string]string{"X-Auth-Token": token})
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
func PowerOnHost(host *fastjson.Value, tokens *fastjson.Value, force bool) (string, error) {
	var state string
	if force {
		state = "PushPowerButton"
	} else {
		state = "On"
	}
	return setPowerState(host, tokens, state)
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
		return "", err
	}

	ip := utils.GetConfigFieldValue(host, "ControllerIP")
	url := "https://" + ip + "/redfish/v1/Systems/System.Embedded.1"
	_, body, err := fetch(url, "Get", "", map[string]string{"X-Auth-Token": token})
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