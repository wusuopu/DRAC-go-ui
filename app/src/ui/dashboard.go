package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/VladimirMarkelov/clui"
	term "github.com/nsf/termbox-go"
	"github.com/valyala/fastjson"
	"main.go/src/api"
	"main.go/src/utils"
)

type Dashboard struct {
	loadingLabel *clui.Label
	table *clui.TableView
	checkAllBtn *clui.Button
	loadingDlg *clui.Window

	data *fastjson.Value
	tokens *fastjson.Value
	selectedRows []int
	totalCount int
}


func confirm(title string, message string, onOk func()) {
	dlg := clui.CreateConfirmationDialog(title, message, []string{"Ok", "Cancel"}, 2)
	dlg.OnClose(func() {
		if dlg.Result() == 1 {
			onOk()
		}
	})
}

func (d *Dashboard) initDashboard() {
	view := clui.AddWindow(0, 0, 40, 30, "Dell Remote Access Controller")
	view.SetBackColor(term.ColorBlack)
	view.SetTextColor(term.ColorWhite)
	view.SetMaximized(true)
	// view.SetTitleButtons(clui.ButtonMaximize | clui.ButtonClose)
	view.SetPack(clui.Vertical)
	view.SetGaps(0, 1)
	view.OnKeyDown(func(e clui.Event, i interface{}) bool {
		if e.Ch == 'q' || e.Ch == 'Q' {
			go clui.Stop()
		}
		if e.Ch =='f' || e.Ch == 'F' {
			d.forcePowerOff(d.table.SelectedRow())
		}
		return false
	}, nil)


	topFrame := clui.CreateFrame(view, clui.AutoSize, clui.AutoSize, clui.BorderThick, clui.Fixed)
	topFrame.SetPack(clui.Vertical)
	topFrame.SetGaps(10, 0)
	topFrame.SetTitle("使用说明")

	clui.CreateLabel(topFrame, 65, 1, "程序每隔10秒刷新一次状态;", clui.Fixed)
	clui.CreateLabel(topFrame, 65, 1, "鼠标点击按钮进行相关操作;按 Q 键退出程序;", clui.Fixed)
	clui.CreateLabel(topFrame, 65, 1, "使用下上方向键移动表格的光标,按 空格键 选中当前的光标的主机;", clui.Fixed)
	clui.CreateLabel(topFrame, 65, 1, "若机器不能正常关机，则按 F 键强制关机当前光标的机器;", clui.Fixed)

	toolbarFram := clui.CreateFrame(view, clui.AutoSize, clui.AutoSize, clui.BorderThick, clui.Fixed)
	toolbarFram.SetGaps(10, 0)
	d.checkAllBtn = clui.CreateButton(toolbarFram, 20, 1, "全选/反选(0/0)", clui.Fixed)
	d.checkAllBtn.OnClick(d.toogleCheckAll)

	powerOffButton := clui.CreateButton(toolbarFram, 20, 1, "批量关机", clui.Fixed)
	powerOffButton.OnClick(d.batchPowerOff)
	powerOnButton := clui.CreateButton(toolbarFram, 20, 1, "批量开机", clui.Fixed)
	powerOnButton.OnClick(d.batchPowerOn)

	quitButton := clui.CreateButton(toolbarFram, 6, 4, "退出", clui.Fixed)
	quitButton.SetShadowType(clui.ShadowHalf)
	quitButton.OnClick(func(e clui.Event) {
		go clui.Stop()
	})
	quitButton.SetTextColor(term.ColorBlack)
	quitButton.SetPaddings(20, 4)

	d.table = clui.CreateTableView(view, 20, 30, clui.AutoSize)
	d.table.SetShowLines(true)
	d.table.SetFullRowSelect(true)
	headers := []string{"", "ID", "HostName", "ControllerIP", "Network Stat", "Power Stat"}
	cols:= []clui.Column{
		clui.Column{Title: "Checked", Width: 5, Alignment: clui.AlignLeft},
		clui.Column{Title: "ID", Width: 5, Alignment: clui.AlignLeft},
		clui.Column{Title: "HostName", Width: 15, Alignment: clui.AlignCenter},
		clui.Column{Title: "ControllerIP", Width: 20, Alignment: clui.AlignCenter},
		clui.Column{Title: "Network Stat", Width: 15, Alignment: clui.AlignCenter},
		clui.Column{Title: "Power Stat", Width: 15, Alignment: clui.AlignRight},
	}
	d.table.SetColumns(cols)

	d.table.OnKeyPress(func(k term.Key) bool {
		row := d.table.SelectedRow()
		switch k {
		case term.KeySpace:
			d.toggleSelection(row)
			return true
		}
		return false
	})
	d.table.OnDrawCell(func(info *clui.ColumnDrawInfo) {
		// clui.Logger().Printf("OnDrawCell: %d %d", info.Row, info.Col)
		if d.data == nil {
			return
		}
		row := strconv.Itoa(info.Row)
		name := headers[info.Col]
		if (d.data.Exists(row, name)) {
			info.Text = strings.Trim(d.data.Get(row, name).String(), "\"")
		}

		checked := slices.Contains(d.selectedRows, info.Row)
		if info.Col == 0 && checked {
			info.Text = "*"
		}
		if checked {
			info.Bg = term.ColorYellow
			info.Fg = term.ColorBlack
		}
		if info.RowSelected {
			info.Bg = term.ColorGreen
			info.Fg = term.ColorBlack
		}
	})

	d.loadingLabel = clui.CreateLabel(toolbarFram, 40, clui.AutoSize, "状态刷新中 (0/0)...", clui.Fixed)

	clui.ActivateControl(view, d.table)

	cw, ch := term.Size()
	d.loadingDlg = clui.AddWindow(cw/2-12, ch/2-8, 30, 3, "Loading...")
	d.loadingDlg.SetModal(true)
	d.loadingDlg.SetVisible(false)
}
func (d *Dashboard) updateCheckAllButtonText () {
	d.checkAllBtn.SetTitle(fmt.Sprintf("全选/反选(%d/%d)", len(d.selectedRows), d.totalCount))
}
func (d *Dashboard) toggleSelection(row int) {
	if slices.Contains(d.selectedRows, row) {
		d.selectedRows = slices.DeleteFunc(d.selectedRows, func(i int) bool { return i == row })
	} else {
		d.selectedRows = append(d.selectedRows, row)
	}
	d.updateCheckAllButtonText()
}
func (d *Dashboard) toogleCheckAll(e clui.Event) {
	if len(d.selectedRows) == d.totalCount {
		// 清空选择
		d.selectedRows = []int{}
	} else {
		// 全选
		d.selectedRows = []int{}
		for i := 0; i < d.totalCount; i++ {
			d.selectedRows = append(d.selectedRows, i)
		}
	}
	d.updateCheckAllButtonText()
	d.Refresh()
}
func (d *Dashboard) valid() bool {
	if len(d.selectedRows) == 0 {
		clui.CreateAlertDialog("提示", "Please select one host at least", "OK")
		return false
	}
	return true
}
func (d *Dashboard) batchPowerOff(e clui.Event) {
	if !d.valid() { return }
	confirm("确认", "Confirm again.", func() {
		d.loadingDlg.SetVisible(true)
		defer d.loadingDlg.SetVisible(false)
		for _, v := range d.selectedRows {
			host := d.data.Get(strconv.Itoa(v))
			state := utils.GetConfigFieldValue(host, "Power Stat")
			if state != "On" {
				continue
			}
			clui.Logger().Printf("power off %s", host.Get("HostName").String())
			api.PowerOffHost(host, d.tokens, false)
		}
	})
}
func (d *Dashboard) batchPowerOn(e clui.Event) {
	if !d.valid() { return }
	confirm("确认", "Confirm again.", func() {
		d.loadingDlg.SetVisible(true)
		defer d.loadingDlg.SetVisible(false)
		for _, v := range d.selectedRows {
			host := d.data.Get(strconv.Itoa(v))
			state := utils.GetConfigFieldValue(host, "Power Stat")
			if state == "On" {
				continue
			}
			clui.Logger().Printf("power on %s", host.Get("HostName").String())
			api.PowerOnHost(host, d.tokens)
		}
	})
}
func (d *Dashboard) forcePowerOff(row int) {
	name := d.data.Get(strconv.Itoa(row), "HostName").String()
	confirm("确认", "Confirm again.\nPower Off " + name, func() {
		d.loadingDlg.SetVisible(true)
		defer d.loadingDlg.SetVisible(false)
		host := d.data.Get(strconv.Itoa(row))
		api.PowerOffHost(host, d.tokens, true)
	})
}

func (d *Dashboard) SetData(data *fastjson.Value) {
	d.data = data
	d.totalCount = len(data.GetArray())
	d.table.SetRowCount(d.totalCount)
	d.updateCheckAllButtonText()
}
func (d *Dashboard) SetTokens(tokens *fastjson.Value) {
	d.tokens = tokens
}
func (d *Dashboard) GetData() *fastjson.Value {
	return d.data
}
func (d *Dashboard) GetTokns() *fastjson.Value {
	return d.tokens
}
func (d *Dashboard) SetProgress(totalCount uint, current uint) {
	if totalCount == current {
		d.loadingLabel.SetTitle(fmt.Sprintf("状态更新完成(%s)", time.Now().Format("15:04:05")))
	} else {
		d.loadingLabel.SetTitle(fmt.Sprintf("状态刷新中 (%d/%d)...", current, totalCount))
	}
}
func (d *Dashboard) Refresh() {
	clui.RefreshScreen()
}

func (d *Dashboard) Run(init func (d *Dashboard)) {
	clui.InitLibrary()
	defer clui.DeinitLibrary()

	d.initDashboard()
	
	init(d)

	clui.MainLoop()
}