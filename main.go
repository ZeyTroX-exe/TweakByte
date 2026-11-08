package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"
)

var (
	PIDs  = []string{}
	ADDRs = []string{}

	user32        = syscall.NewLazyDLL("user32.dll")
	getWindowText = user32.NewProc("GetWindowTextW")

	addresses = []uintptr{}

	selectedAddress uintptr
	selectedPid     uint32
)

func getWindows() map[string]uint32 {
	windowMap := map[string]uint32{}

	windows.EnumWindows(syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if !windows.IsWindowVisible(windows.HWND(hwnd)) {
			return 1
		}
		var titleBuffer [256]uint16
		getWindowText.Call(hwnd, uintptr(unsafe.Pointer(&titleBuffer[0])), 256)

		title := syscall.UTF16ToString(titleBuffer[:])
		if title != "" {
			var pid uint32
			windows.GetWindowThreadProcessId(windows.HWND(hwnd), &pid)
			windowMap[title] = pid
		}
		return 1
	}), nil)

	return windowMap
}

func writeToAddress(addr uintptr, pid, value uint32) error {
	handle, _ := windows.OpenProcess(windows.PROCESS_VM_WRITE|windows.PROCESS_VM_OPERATION|windows.PROCESS_QUERY_INFORMATION, false, pid)
	defer windows.CloseHandle(handle)

	err := windows.WriteProcessMemory(handle, addr, (*byte)(unsafe.Pointer(&value)), unsafe.Sizeof(value), nil)
	return err
}

func readFromAddress(addr uintptr, pid uint32) (uint32, error) {
	handle, _ := windows.OpenProcess(windows.PROCESS_VM_READ|windows.PROCESS_VM_OPERATION|windows.PROCESS_QUERY_INFORMATION, false, pid)
	defer windows.CloseHandle(handle)

	var value uint32
	err := windows.ReadProcessMemory(handle, addr, (*byte)(unsafe.Pointer(&value)), unsafe.Sizeof(value), nil)
	return value, err
}

func getValues() {
	PIDs = []string{}
	ADDRs = []string{}

	for _, addr := range addresses {
		ADDRs = append(ADDRs, strconv.FormatUint(uint64(addr), 16))
	}

	for name, pid := range getWindows() {
		PIDs = append(PIDs, fmt.Sprintf("%v :: %v", strconv.Itoa(int(pid)), name))
	}
}

func main() {
	myApp := app.New()

	myWin := myApp.NewWindow("TweakByte")
	myWin.Resize(fyne.NewSize(200, 300))
	myWin.SetFixedSize(true)
	myWin.SetIcon(theme.DocumentCreateIcon())

	getValues()

	// select a process
	procSelect := widget.NewSelect(PIDs, func(nameId string) {
		pidStr := strings.TrimSpace(strings.Split(nameId, "::")[0])
		pid, _ := strconv.Atoi(pidStr)

		selectedPid = uint32(pid)
	})

	procSelect.PlaceHolder = "Select a Process..."

	newValueEntry := widget.NewEntry()
	newValueEntry.PlaceHolder = "New Value..."

	// select from saved addresses
	addrSelect := widget.NewSelect(ADDRs, func(addr string) {
		addrI, _ := strconv.ParseUint(addr, 16, 64)
		selectedAddress = uintptr(addrI)
		value, _ := readFromAddress(uintptr(addrI), selectedPid)

		newValueEntry.Text = strconv.Itoa(int(value))
		newValueEntry.Refresh()
	})

	addrSelect.PlaceHolder = "Saved Addresses..."

	addrEntry := widget.NewEntry()
	addrEntry.PlaceHolder = "Memory Address..."

	// remove saved address
	rmAddr := widget.NewButton("Remove", func() {
		addr := addrEntry.Text
		addrI, _ := strconv.ParseUint(addr, 16, 64)

		for index, addrs := range addresses {
			if addrs == uintptr(addrI) {
				addresses = append(addresses[:index], addresses[index+1:]...)
				break
			}
		}

		getValues()
		addrSelect.ClearSelected()
		procSelect.Options = PIDs
		addrSelect.Options = ADDRs
		procSelect.Refresh()
		addrSelect.Refresh()
	})

	// save address
	addAddr := widget.NewButton("Add", func() {
		addr := addrEntry.Text
		addrI, _ := strconv.ParseUint(addr, 16, 64)

		if addrI != 0 {
			addresses = append(addresses, uintptr(addrI))
		}

		getValues()
		addrSelect.ClearSelected()
		procSelect.Options = PIDs
		addrSelect.Options = ADDRs
		procSelect.Refresh()
		addrSelect.Refresh()
	})

	// overwrite address
	overwriteAddr := widget.NewButton("Overwrite", func() {
		newVal, _ := strconv.Atoi(newValueEntry.Text)
		writeToAddress(selectedAddress, selectedPid, uint32(newVal))
	})

	text := canvas.NewText("TweakByte", color.RGBA{200, 200, 200, 255})
	text.Alignment = fyne.TextAlignCenter
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = 25

	myWin.SetContent(container.NewCenter(container.NewVBox(text, widget.NewCard("", "", container.NewVBox(procSelect, addrSelect, addrEntry, rmAddr, addAddr)), widget.NewCard("", "", container.NewVBox(newValueEntry, overwriteAddr)))))
	myWin.ShowAndRun()
}
