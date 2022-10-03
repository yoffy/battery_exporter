package battery

import (
	"syscall"
	"unsafe"
)

type Handle struct {
	BatteryHandle syscall.Handle
	BatteryTag    uint32
}

// winddk.h
const (
	FILE_READ_DATA  = 0x00000001
	FILE_WRITE_DATA = 0x00000002
)

// batclass.h
var (
	// a.k.a. GUID_DEVICE_BATTERY
	GUID_DEVCLASS_BATTERY = Guid{0x72631E54, 0x78A4, 0x11D0, [8]byte{0xBC, 0xF7, 0x00, 0xAA, 0x00, 0xB7, 0xB3, 0x2A}}
)

type BATTERY_INFORMATION struct {
	Capabilities        int32
	Technology          byte
	Reserved            [3]byte
	Chemistry           [4]byte
	DesignedCapacity    int32
	FullChargedCapacity int32
	DefaultAlert1       int32
	DefaultAlert2       int32
	CriticalBias        int32
	CycleCount          int32
}

type BATTERY_STATUS struct {
	PowerState uint32 // 1: power on line, 2: discharging, 4: charging, 8: critical
	Capacity   uint32 // milli Wh, percent = Capacity / FullChargedCapacity
	Voltage    uint32 // milli Volt
	Rate       int32  // milli Watt
}

// InformationLevel
const (
	BatteryInformation            = 0
	BatteryGranularityInformation = 1
	BatteryTemperature            = 2
	BatteryEstimatedTime          = 3
	BatteryDeviceName             = 4
	BatteryManufactureDate        = 5
	BatteryManufactureName        = 6
	BatteryUniqueID               = 7
	BatterySerialNumber           = 8
)

type BATTERY_QUERY_INFORMATION struct {
	BatteryTag       uint32
	InformationLevel int32
	AtRate           int32
}

type BATTERY_WAIT_STATUS struct {
	BatteryTag   uint32
	Timeout      uint32
	PowerState   uint32
	LowCapacity  uint32
	HighCapacity uint32
}

const (
	IOCTL_BATTERY_QUERY_TAG         = uint32((0x29 << 16) | (1 << 14) | (0x10 << 2))
	IOCTL_BATTERY_QUERY_INFORMATION = uint32((0x29 << 16) | (1 << 14) | (0x11 << 2))
	IOCTL_BATTERY_QUERY_STATUS      = uint32((0x29 << 16) | (1 << 14) | (0x13 << 2))
)

// 10ths of a degree Kelvin to Celcius
func TemperatureToCelcius(k10 uint32) float64 {
	return (float64(k10) - 2731.6) / 10.0
}

func getBatteryHandle(index uint32) (syscall.Handle, error) {
	deviceHandle, err := SetupDiGetClassDevs(
		&GUID_DEVCLASS_BATTERY,
		"",
		0,
		DIGCF_PRESENT|DIGCF_DEVICEINTERFACE)
	if err != nil {
		return syscall.Handle(0), err
	}

	deviceInterfaceData := SP_DEVICE_INTERFACE_DATA{}
	deviceInterfaceData.CbSize = uint32(unsafe.Sizeof(deviceInterfaceData))
	deviceDetailData := SP_DEVICE_INTERFACE_DETAIL_DATA{}
	deviceDetailData.CbSize = 8 // Sizeof(padded struct{DWORD, [1]WCHAR})

	err = SetupDiEnumDeviceInterfaces(
		deviceHandle,
		0,
		&GUID_DEVCLASS_BATTERY,
		index,
		&deviceInterfaceData)
	if err != nil {
		return syscall.Handle(0), err
	}

	err = SetupDiGetDeviceInterfaceDetail(
		deviceHandle,
		&deviceInterfaceData,
		&deviceDetailData, uint32(unsafe.Sizeof(deviceDetailData)),
		nil,
		0)
	if err != nil {
		return syscall.Handle(0), err
	}

	batteryHandle, err := syscall.CreateFile(
		(*uint16)(unsafe.Pointer(&deviceDetailData.DevicePath[0])),
		FILE_READ_DATA|FILE_WRITE_DATA,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0)
	if err != nil {
		return syscall.Handle(0), err
	}

	return batteryHandle, nil
}

func getBatteryTag(batteryHandle syscall.Handle) (uint32, error) {
	timeoutMs := uint32(0)
	tag := uint32(0)
	err := syscall.DeviceIoControl(
		batteryHandle,
		IOCTL_BATTERY_QUERY_TAG,
		(*byte)(unsafe.Pointer(&timeoutMs)), uint32(unsafe.Sizeof(timeoutMs)),
		(*byte)(unsafe.Pointer(&tag)), uint32(unsafe.Sizeof(tag)),
		nil,
		nil)
	if err != nil {
		return 0, err
	}

	return tag, nil
}

func OpenBatteryHandle(index int) (*Handle, error) {
	handle, err := getBatteryHandle(uint32(index))
	if err != nil {
		return nil, err
	}

	tag, err := getBatteryTag(handle)
	if err != nil {
		syscall.CloseHandle(handle)
		return nil, err
	}

	return &Handle{
		BatteryHandle: handle,
		BatteryTag:    tag,
	}, nil
}

func CloseBatteryHandle(handle *Handle) {
	syscall.CloseHandle(handle.BatteryHandle)
}

func GetBatteryInfo(handle *Handle) (*BATTERY_INFORMATION, error) {
	query := BATTERY_QUERY_INFORMATION{handle.BatteryTag, BatteryInformation, 0}
	info := BATTERY_INFORMATION{}
	err := syscall.DeviceIoControl(
		handle.BatteryHandle,
		IOCTL_BATTERY_QUERY_INFORMATION,
		(*byte)(unsafe.Pointer(&query)), uint32(unsafe.Sizeof(query)),
		(*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)),
		nil,
		nil)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func GetBatteryTemperature(handle *Handle) (float64, error) {
	query := BATTERY_QUERY_INFORMATION{handle.BatteryTag, BatteryTemperature, 0}
	temp := uint32(0)
	err := syscall.DeviceIoControl(
		handle.BatteryHandle,
		IOCTL_BATTERY_QUERY_INFORMATION,
		(*byte)(unsafe.Pointer(&query)), uint32(unsafe.Sizeof(query)),
		(*byte)(unsafe.Pointer(&temp)), uint32(unsafe.Sizeof(temp)),
		nil,
		nil)
	if err != nil {
		return 0.0, err
	}

	return TemperatureToCelcius(temp), nil
}

func escape(name []uint16, length int) []uint16 {
	id := make([]uint16, 0)
	for i := 0; i < length; i++ {
		c := name[i]
		// this method is broken as hexadecimal, but works
		// ex. 0x1A -> "%1:"
		if c < 0x20 {
			id = append(id, uint16('%'), uint16('0'+c/16), uint16('0'+c%16))
		} else {
			id = append(id, c)
		}
	}
	return append(id, 0)
}

func GetBatteryUniqueId(handle *Handle) (string, error) {
	query := BATTERY_QUERY_INFORMATION{handle.BatteryTag, BatteryUniqueID, 0}
	name := make([]uint16, 256)
	numOutBytes := uint32(0)
	err := syscall.DeviceIoControl(
		handle.BatteryHandle,
		IOCTL_BATTERY_QUERY_INFORMATION,
		(*byte)(unsafe.Pointer(&query)), uint32(unsafe.Sizeof(query)),
		(*byte)(unsafe.Pointer(&name[0])), uint32(len(name)*2),
		&numOutBytes,
		nil)
	if err != nil {
		return "", err
	}

	id := escape(name, int(numOutBytes/2)-1)

	return syscall.UTF16ToString(id), nil
}

func GetBatteryStatus(handle *Handle) (*BATTERY_STATUS, error) {
	query := BATTERY_WAIT_STATUS{handle.BatteryTag, 0, 0, 0, 0}
	status := BATTERY_STATUS{}
	err := syscall.DeviceIoControl(
		handle.BatteryHandle,
		IOCTL_BATTERY_QUERY_STATUS,
		(*byte)(unsafe.Pointer(&query)), uint32(unsafe.Sizeof(query)),
		(*byte)(unsafe.Pointer(&status)), uint32(unsafe.Sizeof(status)),
		nil,
		nil)
	if err != nil {
		return nil, err
	}

	return &status, nil
}
