package server

import (
	"database/sql"
	"fmt"
)

type PortSettings struct {
	SSHPort   int `json:"sshPort"`
	RDPPort   int `json:"rdpPort"`
	SMBPort   int `json:"smbPort"`
	HTTPPort  int `json:"httpPort"`
	HTTPSPort int `json:"httpsPort"`
	WinRMPort int `json:"winrmPort"`
}

func GetPortSettings(db *sql.DB) PortSettings {
	s := PortSettings{SSHPort: 22, RDPPort: 3389, SMBPort: 445, HTTPPort: 80, HTTPSPort: 443, WinRMPort: 5985}
	db.QueryRow(`SELECT ssh_port,rdp_port,smb_port,http_port,COALESCE(https_port,443),winrm_port FROM port_settings WHERE id=1`).
		Scan(&s.SSHPort, &s.RDPPort, &s.SMBPort, &s.HTTPPort, &s.HTTPSPort, &s.WinRMPort)
	return s
}

func SavePortSettings(db *sql.DB, s PortSettings) error {
	_, err := db.Exec(`INSERT INTO port_settings (id,ssh_port,rdp_port,smb_port,http_port,https_port,winrm_port)
		VALUES (1,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		  ssh_port=excluded.ssh_port, rdp_port=excluded.rdp_port,
		  smb_port=excluded.smb_port, http_port=excluded.http_port,
		  https_port=excluded.https_port, winrm_port=excluded.winrm_port`,
		s.SSHPort, s.RDPPort, s.SMBPort, s.HTTPPort, s.HTTPSPort, s.WinRMPort)
	return err
}

// NodePortOverride — переопределения портов для конкретного узла.
// nil-поле означает «использовать глобальный дефолт».
type NodePortOverride struct {
	SSHPort   *int `json:"sshPort"`
	RDPPort   *int `json:"rdpPort"`
	SMBPort   *int `json:"smbPort"`
	HTTPPort  *int `json:"httpPort"`
	HTTPSPort *int `json:"httpsPort"`
	WinRMPort *int `json:"winrmPort"`
}

func GetNodePortOverride(db *sql.DB, nodeName string) NodePortOverride {
	var o NodePortOverride
	db.QueryRow(`SELECT ssh_port,rdp_port,smb_port,http_port,https_port,winrm_port
		FROM node_port_overrides WHERE node_name=?`, nodeName).
		Scan(&o.SSHPort, &o.RDPPort, &o.SMBPort, &o.HTTPPort, &o.HTTPSPort, &o.WinRMPort)
	return o
}

func SaveNodePortOverride(db *sql.DB, nodeName string, o NodePortOverride) error {
	_, err := db.Exec(`INSERT INTO node_port_overrides
		(node_name,ssh_port,rdp_port,smb_port,http_port,https_port,winrm_port)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(node_name) DO UPDATE SET
		  ssh_port=excluded.ssh_port, rdp_port=excluded.rdp_port,
		  smb_port=excluded.smb_port, http_port=excluded.http_port,
		  https_port=excluded.https_port, winrm_port=excluded.winrm_port`,
		nodeName, o.SSHPort, o.RDPPort, o.SMBPort, o.HTTPPort, o.HTTPSPort, o.WinRMPort)
	return err
}

// EffectivePortSettings возвращает итоговые порты для узла.
func EffectivePortSettings(global PortSettings, override NodePortOverride) PortSettings {
	s := global
	if override.SSHPort != nil   { s.SSHPort   = *override.SSHPort }
	if override.RDPPort != nil   { s.RDPPort   = *override.RDPPort }
	if override.SMBPort != nil   { s.SMBPort   = *override.SMBPort }
	if override.HTTPPort != nil  { s.HTTPPort  = *override.HTTPPort }
	if override.HTTPSPort != nil { s.HTTPSPort = *override.HTTPSPort }
	if override.WinRMPort != nil { s.WinRMPort = *override.WinRMPort }
	return s
}

// CustomService — пользовательский сервис для TCP-пробы
type CustomService struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Port int    `json:"port"`
}

// CustomServiceResult — результат пробы пользовательского сервиса
type CustomServiceResult struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Port      int     `json:"port"`
	Reachable bool    `json:"reachable"`
	Ms        float64 `json:"ms"`
}

func GetCustomServices(db *sql.DB) []CustomService {
	rows, err := db.Query(`SELECT id, name, port FROM custom_services ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []CustomService
	for rows.Next() {
		var s CustomService
		if rows.Scan(&s.ID, &s.Name, &s.Port) == nil {
			result = append(result, s)
		}
	}
	return result
}

func CreateCustomService(db *sql.DB, name string, port int) (CustomService, error) {
	res, err := db.Exec(`INSERT INTO custom_services(name, port) VALUES(?, ?)`, name, port)
	if err != nil {
		return CustomService{}, err
	}
	id, _ := res.LastInsertId()
	return CustomService{ID: int(id), Name: name, Port: port}, nil
}

func DeleteCustomService(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM custom_services WHERE id=?`, id)
	db.Exec(`DELETE FROM node_service_visibility WHERE service_key=?`, fmt.Sprintf("custom_%d", id))
	db.Exec(`DELETE FROM node_custom_service_ports WHERE service_id=?`, id)
	return err
}

// GetNodeCustomServicePorts возвращает per-node переопределения портов кастомных сервисов.
// Ключ — ID кастомного сервиса, значение — порт.
func GetNodeCustomServicePorts(db *sql.DB, nodeName string) map[int]int {
	rows, err := db.Query(`SELECT service_id, port FROM node_custom_service_ports WHERE node_name=?`, nodeName)
	if err != nil {
		return map[int]int{}
	}
	defer rows.Close()
	result := map[int]int{}
	for rows.Next() {
		var id, port int
		if rows.Scan(&id, &port) == nil {
			result[id] = port
		}
	}
	return result
}

// SaveNodeCustomServicePorts сохраняет per-node переопределения портов кастомных сервисов.
// ports — карта {serviceId: port}; nil/пустая карта очищает все переопределения.
func SaveNodeCustomServicePorts(db *sql.DB, nodeName string, ports map[int]int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	tx.Exec(`DELETE FROM node_custom_service_ports WHERE node_name=?`, nodeName)
	for id, port := range ports {
		tx.Exec(`INSERT INTO node_custom_service_ports(node_name, service_id, port) VALUES(?,?,?)`, nodeName, id, port)
	}
	return tx.Commit()
}

// NodeServiceVisibility — карта видимости сервисов для конкретного узла.
// Ключи: "ssh", "rdp", "smb", "http", "winrm", "custom_N"
// Если ключа нет — сервис считается видимым (дефолт).
type NodeServiceVisibility map[string]bool

func GetNodeServiceVisibility(db *sql.DB, nodeName string) NodeServiceVisibility {
	rows, err := db.Query(`SELECT service_key, visible FROM node_service_visibility WHERE node_name=?`, nodeName)
	if err != nil {
		return NodeServiceVisibility{}
	}
	defer rows.Close()
	result := NodeServiceVisibility{}
	for rows.Next() {
		var key string
		var visible int
		if rows.Scan(&key, &visible) == nil {
			result[key] = visible != 0
		}
	}
	return result
}

func SaveNodeServiceVisibility(db *sql.DB, nodeName string, vis NodeServiceVisibility) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	tx.Exec(`DELETE FROM node_service_visibility WHERE node_name=?`, nodeName)
	for key, visible := range vis {
		v := 0
		if visible {
			v = 1
		}
		tx.Exec(`INSERT INTO node_service_visibility(node_name, service_key, visible) VALUES(?,?,?)`, nodeName, key, v)
	}
	return tx.Commit()
}
