package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gosnmp/gosnmp"
)

type SNMPResult struct {
	DeviceID   uint      `json:"device_id"`
	DeviceName string    `json:"device_name"`
	IPAddress  string    `json:"ip_address"`
	OID        string    `json:"oid"`
	OIDName    string    `json:"oid_name"`
	Value      string    `json:"value"`
	ValueType  string    `json:"value_type"`
	CheckedAt  time.Time `json:"checked_at"`
}

var CommonOIDs = map[string]string{
	"1.3.6.1.2.1.1.1.0": "sysDescr",
	"1.3.6.1.2.1.1.3.0": "sysUpTime",
	"1.3.6.1.2.1.1.4.0": "sysContact",
	"1.3.6.1.2.1.1.5.0": "sysName",
	"1.3.6.1.2.1.1.6.0": "sysLocation",
	"1.3.6.1.2.1.2.1.0": "ifNumber",
	"1.3.6.1.2.1.25.3.3.1.2.196608": "hrProcessorLoad",
	"1.3.6.1.2.1.25.2.3.1.6.1":      "hrStorageUsed",
	"1.3.6.1.2.1.25.2.3.1.5.1":      "hrStorageSize",
}

type SNMPService struct {
	ch driver.Conn
	db string
}

func NewSNMPService(ch driver.Conn, db string) *SNMPService {
	return &SNMPService{ch: ch, db: db}
}

func (s *SNMPService) Walk(deviceID uint, deviceName, ip, community, version string, port int, oids []string, checkedBy uint) ([]SNMPResult, error) {
	if port == 0 {
		port = 161
	}
	if community == "" {
		community = "public"
	}

	snmpVersion := gosnmp.Version2c
	switch version {
	case "1":
		snmpVersion = gosnmp.Version1
	case "3":
		snmpVersion = gosnmp.Version3
	}

	client := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(port),
		Community: community,
		Version:   snmpVersion,
		Timeout:   time.Duration(10) * time.Second,
		Retries:   2,
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("gagal koneksi SNMP ke %s: %v", ip, err)
	}
	defer client.Conn.Close()

	if len(oids) == 0 {
		for oid := range CommonOIDs {
			oids = append(oids, oid)
		}
	}

	packets, err := client.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET gagal: %v", err)
	}

	var results []SNMPResult
	now := time.Now()

	for _, variable := range packets.Variables {
		oidName := CommonOIDs[variable.Name]
		if oidName == "" {
			oidName = variable.Name
		}

		value := fmt.Sprintf("%v", variable.Value)
		valueType := variable.Type.String()

		result := SNMPResult{
			DeviceID:   deviceID,
			DeviceName: deviceName,
			IPAddress:  ip,
			OID:        variable.Name,
			OIDName:    oidName,
			Value:      value,
			ValueType:  valueType,
			CheckedAt:  now,
		}
		results = append(results, result)
	}

	if err := s.saveToCH(results, checkedBy); err != nil {
		log.Printf("⚠️  Gagal simpan SNMP result ke ClickHouse: %v", err)
	}

	return results, nil
}

func (s *SNMPService) saveToCH(results []SNMPResult, checkedBy uint) error {
	ctx := context.Background()
	batch, err := s.ch.PrepareBatch(ctx, fmt.Sprintf(`
		INSERT INTO %s.snmp_results
		(device_id, device_name, ip_address, oid, oid_name, value, value_type, checked_by, checked_at)
	`, s.db))
	if err != nil {
		return err
	}

	for _, r := range results {
		if err := batch.Append(r.DeviceID, r.DeviceName, r.IPAddress, r.OID, r.OIDName,
			r.Value, r.ValueType, checkedBy, r.CheckedAt); err != nil {
			return err
		}
	}
	return batch.Send()
}

func (s *SNMPService) GetHistory(deviceID uint, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT id, device_id, device_name, ip_address, oid, oid_name, value, value_type, checked_by, checked_at
		FROM %s.snmp_results
		WHERE device_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`, s.db)

	rows, err := s.ch.Query(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, deviceName, ipAddr, oid, oidName, value, valType string
		var dID, chBy uint64
		var checkedAt time.Time
		if err := rows.Scan(&id, &dID, &deviceName, &ipAddr, &oid, &oidName, &value, &valType, &chBy, &checkedAt); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id": id, "device_id": dID, "device_name": deviceName, "ip_address": ipAddr,
			"oid": oid, "oid_name": oidName, "value": value, "value_type": valType,
			"checked_by": chBy, "checked_at": checkedAt,
		})
	}
	return results, nil
}

func (s *SNMPService) GetAvailableOIDs() map[string]string {
	return CommonOIDs
}
