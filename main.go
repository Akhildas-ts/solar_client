package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ✅ Format 1: Your current format (nested data)
type Format1Payload struct {
	DeviceType     string `json:"device_type"`
	DeviceName     string `json:"device_name"`
	DeviceID       string `json:"device_id"`
	Date           string `json:"date"`
	Time           string `json:"time"`
	SignalStrength string `json:"signal_strength"`
	Data           struct {
		SerialNo         string `json:"serial_no"`
		S1V              int    `json:"s1v"`
		TotalOutputPower int    `json:"total_output_power"`
		F                int    `json:"f"`
		TodayE           int    `json:"today_e"`
		TotalE           int    `json:"total_e"`
		InvTemp          int    `json:"inv_temp"`
		FaultCode        int    `json:"fault_code"`
	} `json:"data"`
}

// ✅ Format 2: Different field names (nested)
type Format2Payload struct {
	DeviceType string `json:"device_type"`
	DeviceName string `json:"device_name"`
	DeviceID   string `json:"device_id"`
	Data       struct {
		SerialNo    string `json:"serial_no"`
		Voltage     int    `json:"voltage_input"`    // ✅ DIFFERENT
		PowerOutput int    `json:"power_watts"`      // ✅ DIFFERENT
		Frequency   int    `json:"freq_hz"`          // ✅ DIFFERENT
		DailyEnergy int    `json:"energy_today_wh"`  // ✅ DIFFERENT
		TotalEnergy int    `json:"energy_total_kwh"` // ✅ DIFFERENT
		Temperature int    `json:"temp_celsius"`
		ErrorCode   int    `json:"error_code"`
	} `json:"data"`
}

// ✅ Format 3: Flat structure (no nested data)
type Format3Payload struct {
	DeviceType  string `json:"device_type"`
	DeviceName  string `json:"device_name"`
	DeviceID    string `json:"device_id"`
	SerialNo    string `json:"serial_no"`
	V           int    `json:"V"`       // ✅ SHORT NAME
	P           int    `json:"P"`       // ✅ SHORT NAME
	Hz          int    `json:"Hz"`      // ✅ SHORT NAME
	EnergyDaily int    `json:"E_today"` // ✅ DIFFERENT
	EnergyTotal int    `json:"E_total"` // ✅ DIFFERENT
	Temp        int    `json:"temp"`
	Status      int    `json:"status"`
}

// ✅ Format 4: Mixed with units in field names
type Format4Payload struct {
	DeviceType string `json:"device_type"`
	DeviceName string `json:"device_name"`
	Data       struct {
		VoltageMillivolts int     `json:"voltage_mv"` // ✅ IN MILLIVOLTS!
		PowerKilowatts    float64 `json:"power_kw"`   // ✅ IN KILOWATTS!
		FreqHz            int     `json:"frequency_hz"`
		TodayKwh          float64 `json:"today_kwh"` // ✅ IN KWH!
		TotalKwh          float64 `json:"total_kwh"` // ✅ IN KWH!
		TempFahrenheit    int     `json:"temp_f"`    // ✅ FAHRENHEIT!
		FaultStatus       int     `json:"fault"`
	} `json:"readings"`
}

var totalSent uint64
var formatCounts [4]uint64 // Track sends per format
func main() {
	endpoint := "http://localhost:8080/api/data"
	rate := 600
	runDuration := 15 * time.Minute
	totalRecords := rate * int(runDuration.Seconds())

	fmt.Printf("🚀 Starting multi-format inverter simulator\n")
	fmt.Printf("   Sending %d records/sec across 4 formats\n", rate)
	fmt.Printf("   Target: %d total records in %v\n\n", totalRecords, runDuration)

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        2000,
			MaxIdleConnsPerHost: 2000,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var wg sync.WaitGroup
	startTime := time.Now()
	endTime := startTime.Add(runDuration)

	var failed uint64

	seconds := 0
	for time.Now().Before(endTime) {
		secondStart := time.Now()

		// A) Exact data count (strict 600/sec)
		for i := 0; i < rate; i++ {
			formatType := (seconds*rate + i) % 4
			wg.Add(1)
			go func(format int) {
				defer wg.Done()
				ok := sendFormat(client, endpoint, format)
				if ok {
					atomic.AddUint64(&totalSent, 1)
					atomic.AddUint64(&formatCounts[format], 1)
				} else {
					atomic.AddUint64(&failed, 1)
				}
			}(formatType)
		}

		seconds++

		// Sleep the remainder of the second to stay perfectly aligned
		elapsed := time.Since(secondStart)
		if elapsed < time.Second {
			time.Sleep(time.Second - elapsed)
		}
	}

	wg.Wait()

	elapsed := time.Since(startTime)
	fmt.Printf("\n✅ Finished after %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("   Total Sent: %d | Failed: %d\n", totalSent, failed)
	fmt.Printf("   Actual rate: %.2f/sec\n", float64(totalSent)/elapsed.Seconds())
	for i := 0; i < 4; i++ {
		fmt.Printf("   Format %d: %d\n", i+1, atomic.LoadUint64(&formatCounts[i]))
	}
	//b- stable 
	// start := time.Now()
	// endTime := start.Add(runDuration)
	// ticker := time.NewTicker(time.Second / time.Duration(rate))
	// defer ticker.Stop()

	// var failed uint64

	// // stats printer
	// go func() {
	// 	for range time.NewTicker(10 * time.Second).C {
	// 		total := atomic.LoadUint64(&totalSent)
	// 		fmt.Printf("📊 Sent=%d | Failed=%d | Rate=%.2f/s\n",
	// 			total, atomic.LoadUint64(&failed),
	// 			float64(total)/time.Since(start).Seconds())
	// 		if time.Now().After(endTime) {
	// 			return
	// 		}
	// 	}
	// }()

	// for time.Now().Before(endTime) {
	// 	<-ticker.C
	// 	formatType := int(atomic.LoadUint64(&totalSent) % 4)

	// 	wg.Add(1)
	// 	go func(format int) {
	// 		defer wg.Done()
	// 		ok := sendFormat(client, endpoint, format)
	// 		if ok {
	// 			atomic.AddUint64(&totalSent, 1)
	// 			atomic.AddUint64(&formatCounts[format], 1)
	// 		} else {
	// 			atomic.AddUint64(&failed, 1)
	// 		}
	// 	}(formatType)
	// }

	// wg.Wait()
	// elapsed := time.Since(start)

	// fmt.Printf("\n✅ Done after %v\n", elapsed.Round(time.Millisecond))
	// fmt.Printf("   Sent: %d | Failed: %d | Actual rate: %.2f/sec\n",
	// 	totalSent, failed, float64(totalSent)/elapsed.Seconds())
	// for i := 0; i < 4; i++ {
	// 	fmt.Printf("   Format %d: %d\n", i+1, atomic.LoadUint64(&formatCounts[i]))
	// }
}

func sendFormat(client *http.Client, url string, formatType int) bool {
	now := time.Now()
	deviceNum := rand.Intn(50) + 1

	var payload any
	switch formatType {
	case 0:
		p := Format1Payload{
			DeviceType:     "current_format",
			DeviceName:     fmt.Sprintf("ESIN%d", deviceNum),
			DeviceID:       fmt.Sprintf("ESDL%d", rand.Intn(600)+1),
			Date:           now.Format("02/01/2006"),
			Time:           now.Format("15:04:05"),
			SignalStrength: "-1",
		}
		p.Data.SerialNo = fmt.Sprintf("%d", rand.Intn(600)+1)
		p.Data.S1V = 6200 + rand.Intn(200) - 100
		p.Data.TotalOutputPower = 147000 + rand.Intn(500)
		p.Data.F = 700 + rand.Intn(50)
		p.Data.TodayE = rand.Intn(1000)
		p.Data.TotalE = 500000 + rand.Intn(10000)
		p.Data.InvTemp = 650 + rand.Intn(10) - 5
		p.Data.FaultCode = randomFault()
		payload = p

	case 1:
		p := Format2Payload{
			DeviceType: "format_2_inverter",
			DeviceName: fmt.Sprintf("INV_B_%d", deviceNum),
			DeviceID:   fmt.Sprintf("TYPE_B_%d", rand.Intn(600)+1),
		}
		p.Data.SerialNo = fmt.Sprintf("SN_%d", rand.Intn(600)+1)
		p.Data.Voltage = 6200 + rand.Intn(200) - 100
		p.Data.PowerOutput = 147000 + rand.Intn(500)
		p.Data.Frequency = 700 + rand.Intn(50)
		p.Data.DailyEnergy = rand.Intn(1000)
		p.Data.TotalEnergy = 500 + rand.Intn(100)
		p.Data.Temperature = 65 + rand.Intn(10)
		p.Data.ErrorCode = randomFault()
		payload = p

	case 2:
		p := Format3Payload{
			DeviceType:  "flat_format_device",
			DeviceName:  fmt.Sprintf("FLAT_%d", deviceNum),
			DeviceID:    fmt.Sprintf("FL_%d", rand.Intn(600)+1),
			SerialNo:    fmt.Sprintf("FLAT_SN_%d", rand.Intn(600)+1),
			V:           6200 + rand.Intn(200) - 100,
			P:           147000 + rand.Intn(500),
			Hz:          700 + rand.Intn(50),
			EnergyDaily: rand.Intn(1000),
			EnergyTotal: 500000 + rand.Intn(10000),
			Temp:        650 + rand.Intn(10) - 5,
			Status:      randomFault(),
		}
		payload = p

	case 3:
		p := Format4Payload{
			DeviceType: "unit_conversion_device",
			DeviceName: fmt.Sprintf("CONV_%d", deviceNum),
		}
		voltage := 6200 + rand.Intn(200) - 100
		power := 147000 + rand.Intn(500)
		p.Data.VoltageMillivolts = voltage * 10
		p.Data.PowerKilowatts = float64(power) / 1000
		p.Data.FreqHz = 700 + rand.Intn(50)
		p.Data.TodayKwh = float64(rand.Intn(1000)) / 1000
		p.Data.TotalKwh = float64(500000+rand.Intn(10000)) / 1000
		p.Data.TempFahrenheit = (650+rand.Intn(10)-5)*9/5 + 32
		p.Data.FaultStatus = randomFault()
		payload = p
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("❌ JSON marshal error:", err)
		return false
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("❌ POST error:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("⚠️  Bad response:", resp.Status)
		return false
	}
	return true
}

func randomFault() int {
	if rand.Float64() < 0.1 {
		return rand.Intn(5) + 1
	}
	return 0
}

// key change  i addedd,
// created a diffrent data formate
//
