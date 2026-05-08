package server

import "database/sql"

type PortSettings struct {
	SSHPort   int `json:"sshPort"`
	RDPPort   int `json:"rdpPort"`
	SMBPort   int `json:"smbPort"`
	HTTPPort  int `json:"httpPort"`
	WinRMPort int `json:"winrmPort"`
	DNSPort   int `json:"dnsPort"`
}

func GetPortSettings(db *sql.DB) PortSettings {
	s := PortSettings{SSHPort: 22, RDPPort: 3389, SMBPort: 445, HTTPPort: 80, WinRMPort: 5985, DNSPort: 53}
	db.QueryRow(`SELECT ssh_port,rdp_port,smb_port,http_port,winrm_port,dns_port FROM port_settings WHERE id=1`).
		Scan(&s.SSHPort, &s.RDPPort, &s.SMBPort, &s.HTTPPort, &s.WinRMPort, &s.DNSPort)
	return s
}

func SavePortSettings(db *sql.DB, s PortSettings) error {
	_, err := db.Exec(`INSERT INTO port_settings (id,ssh_port,rdp_port,smb_port,http_port,winrm_port,dns_port)
		VALUES (1,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		  ssh_port=excluded.ssh_port, rdp_port=excluded.rdp_port,
		  smb_port=excluded.smb_port, http_port=excluded.http_port,
		  winrm_port=excluded.winrm_port, dns_port=excluded.dns_port`,
		s.SSHPort, s.RDPPort, s.SMBPort, s.HTTPPort, s.WinRMPort, s.DNSPort)
	return err
}
