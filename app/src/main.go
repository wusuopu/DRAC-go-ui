package main

import (
	"os"
	"strconv"
	"time"

	"github.com/valyala/fastjson"
	"main.go/src/api"
	"main.go/src/ui"
	"main.go/src/utils"
)

func checkState(item *fastjson.Value, tokens *fastjson.Value) {
	// var builder fastjson.Arena
	// state := builder.NewString("OnLine")

	// item.Set("Network Stat", state)
	// item.Set("Power Stat", state)

	// time.Sleep(1 * time.Second)
	api.GetPowerState(item, tokens)
}

func startTick(d *ui.Dashboard) {
	data := d.GetData()
	totalCount := uint(len(data.GetArray()))
	for {
		for i, item := range data.GetArray() {
			checkState(item, d.GetTokns())
			d.SetProgress(uint(totalCount), uint(i+1))
			d.Refresh()
		}
		time.Sleep(10 * time.Second)
	}
}

func initCallback(d *ui.Dashboard) {
	var filename string = ""
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}
	data := utils.LoadConfig(filename)
	tokens := utils.LoadToken("")

	var builder fastjson.Arena
	for i, item := range data.GetArray() {
		item.Set("ID", builder.NewString(strconv.Itoa(i + 1)))

		loading := builder.NewString("检查中...")
		item.Set("Network Stat", loading)
		item.Set("Power Stat", loading)
	}
	d.SetData(data)
	d.SetTokens(tokens)
	go startTick(d)
}

func main() {
	dash := ui.Dashboard{}
	dash.Run(initCallback)
}
