// Package session provides mechanisms to manage and store session information
// and associated device and location details for web applications.
//
// The primary structure of this package is the SessionHub, which serves as a
// central repository for session data (SessInfo) and trusted device/location
// information (DeviceLocationInfo). The package also defines the SessionHubManager
// interface, which outlines methods for adding, removing, listing, and managing
// sessions.
package session

import (
	"encoding/json"
	"time"
)

// HubStore manages the session hub store of a user
type HubStore interface {
	Add(string, SessInfo) error
	Remove(string) error
	List() []SessInfo
	RemoveAll() error
	RemoveExcept(string) error
	ReleaseHubData() error

	// TODO: add functions for managing device location infos
}

type SessInfo struct {
	SessionID string    `json:"sessionID"`
	Exp       time.Time `json:"exp"`

	AgentName    string    `json:"agentName"`
	AgentVersion string    `json:"agentVersion"`
	Device       string    `json:"device"`
	DeviceType   string    `json:"deviceType"`
	Os           string    `json:"os"`
	LastAccessed time.Time `json:"lastAccessed"`

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

type DeviceLocationInfo struct {
	// TODO: Add fields related to trusted device and location information
}

// CurrentHubStoreVersion is used to indicate the current hub store data structure version.
// Change the version to if the current data structure or information is incompatible with the previous one.
// Using this, delete the previous incompatible information before starting the application.
var CurrentHubStoreVersion = "v0.1.0"

type UserSessionHub struct {
	Version             string                        `json:"version"`
	UserID              string                        `json:"userID"`
	SessionStore        map[string]SessInfo           `json:"sessionStore"` // map[sessionID]SessInfo
	DeviceLocationStore map[string]DeviceLocationInfo `json:"deviceLocationStore"`
}

func NewUserSessionHub(userID string, ss map[string]SessInfo, dls map[string]DeviceLocationInfo) *UserSessionHub {
	if ss == nil {
		ss = make(map[string]SessInfo)
	}
	if dls == nil {
		dls = make(map[string]DeviceLocationInfo)
	}

	return &UserSessionHub{
		Version:             CurrentHubStoreVersion,
		UserID:              userID,
		SessionStore:        ss,
		DeviceLocationStore: dls,
	}
}

func NewEmptyUserSessionData(userID string) ([]byte, error) {
	return json.Marshal(NewUserSessionHub(userID, nil, nil))
}
