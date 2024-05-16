package session

import (
	"github.com/oschwald/geoip2-golang"
	"time"
)

type HubStore interface {
	Add(string, SessInfo) error
	Remove(string) error
	List() []SessInfo
	RemoveAll() error
	RemoveExcept(string) error
	ReleaseHubData() error
}

type SessInfo struct {
	SessionID string    `json:"sessionID"`
	Exp       time.Time `json:"-"`

	AgentName    string    `json:"agentName"`
	AgentVersion string    `json:"agentVersion"`
	Device       string    `json:"device"`
	DeviceType   string    `json:"deviceType"`
	Os           string    `json:"os"`
	LastAccessed time.Time `json:"-"`

	GeoLocation GeoLocation `json:"geoLocation"`
	IP          string      `json:"IP"`
}

type GeoLocation struct {
	Continent      string `json:"continent,omitempty"`
	Country        string `json:"country,omitempty"`
	CountryIsoCode string `json:"countryIsoCode,omitempty"`
	City           string `json:"city,omitempty"`
	TimeZone       string `json:"timeZone,omitempty"`
}

type LocationFingerprint string

type DeviceFingerprint string

type TrustedDeviceInfo struct {
}

type HubData struct {
	TrustedDevices   map[DeviceFingerprint]TrustedDeviceInfo
	TrustedLocations map[LocationFingerprint]geoip2.City
	SessInfos        map[string]SessInfo // map[sessionID]SessInfo
}
