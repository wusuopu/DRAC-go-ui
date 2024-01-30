package utils

import (
	"fmt"
	"os"
	"path"

	"github.com/VladimirMarkelov/clui"
	"github.com/valyala/fastjson"
)

/**
  加载配置文件，配置内容： [{HostName: string, ControllerIP: string, username: string, password: string}]
*/
func LoadConfig (f string) *fastjson.Value {
    var filename string = f
    if filename == "" {
        var folder, _ = os.Getwd()
        filename = path.Join(folder, ".config.json")
    }

    var a fastjson.Arena
    var emptyConfig = a.NewArray()
    _, err := os.Stat(filename)
    if err != nil {
        return emptyConfig
    }

    var data, _ = os.ReadFile(filename)
    var p fastjson.Parser
    val, err := p.ParseBytes(data)
    if err != nil {
        fmt.Printf("parse %s error\n", filename)
        return emptyConfig
    }

    if val.Type().String() != "array" {
        return emptyConfig
    }

    return val
}
/**
    加载 token 配置缓存； {[hostname]: {token: string, time: number}}
*/
func LoadToken (f string) *fastjson.Value {
    var filename string = f
    if filename == "" {
        var folder, _ = os.Getwd()
        filename = path.Join(folder, ".token.json")
    }

    var a fastjson.Arena
    var emptyConfig = a.NewObject()
    _, err := os.Stat(filename)
    if err != nil {
        return emptyConfig
    }

    var data, _ = os.ReadFile(filename)
    var p fastjson.Parser
    val, err := p.ParseBytes(data)
    if err != nil {
        fmt.Printf("parse %s error\n", filename)
        return emptyConfig
    }

    if val.Type().String() != "object" {
        return emptyConfig
    }

    return val
}
/**
    保存 token 配置缓存
*/
func SaveToken(f string, tokens *fastjson.Value) {
    var filename string = f
    if filename == "" {
        var folder, _ = os.Getwd()
        filename = path.Join(folder, ".token.json")
    }
    err := os.WriteFile(filename, tokens.MarshalTo(nil), 0666)
    if err != nil {
        clui.Logger().Println("save token error", err)
    }
}
func GetConfigFieldValue(val *fastjson.Value, keys ...string) string {
    if !val.Exists(keys...) {
        return ""
    }
    v := val.Get(keys...)
    if v.Type().String() != "string" {
        return ""
    }

    ret := v.String()
    if len(ret) < 2 { return ret }
    if ret[0] == '"' && ret[len(ret)-1] == '"' {
        // 去掉前后的双引号
        return ret[1:len(ret)-1]
    }
    return ret
}