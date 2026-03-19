package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/persistence"
	"sync"
)

type Device struct {
	Username         string                                `json:"username"`
	Token            string                                `json:"token"`
	IndexerOverrides map[string]config.IndexerSearchConfig `json:"indexer_overrides"`
}

type DeviceManager struct {
	mu      sync.RWMutex
	devices map[string]*Device
	manager *persistence.StateManager
	cfg     *config.Config
	saveFn  func() error
}

var globalDeviceManager *DeviceManager
var deviceManagerMu sync.Mutex

func GetDeviceManager(dataDir string) (*DeviceManager, error) {
	deviceManagerMu.Lock()
	defer deviceManagerMu.Unlock()

	if globalDeviceManager != nil {
		return globalDeviceManager, nil
	}

	manager, err := persistence.GetManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get persistence manager: %w", err)
	}

	dm := &DeviceManager{
		devices: make(map[string]*Device),
		manager: manager,
	}

	if err := dm.load(); err != nil {
		return nil, fmt.Errorf("failed to load devices: %w", err)
	}

	globalDeviceManager = dm
	return dm, nil
}

func NewDeviceManagerFromConfig(cfg *config.Config, saveFn func() error) (*DeviceManager, error) {
	deviceManagerMu.Lock()
	defer deviceManagerMu.Unlock()

	if globalDeviceManager != nil {
		return globalDeviceManager, nil
	}

	if cfg.Devices == nil {
		cfg.Devices = make(map[string]*config.DeviceEntry)
	}

	dm := &DeviceManager{
		devices: make(map[string]*Device),
		cfg:     cfg,
		saveFn:  saveFn,
	}
	if err := dm.load(); err != nil {
		return nil, fmt.Errorf("failed to load devices from config: %w", err)
	}
	globalDeviceManager = dm
	return dm, nil
}

func (dm *DeviceManager) load() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.cfg != nil {
		dm.devices = make(map[string]*Device)
		if dm.cfg.Devices != nil {
			for k, e := range dm.cfg.Devices {
				if e == nil {
					continue
				}
				ov := e.IndexerOverrides
				if ov == nil {
					ov = make(map[string]config.IndexerSearchConfig)
				}
				dm.devices[k] = &Device{
					Username:         e.Username,
					Token:            e.Token,
					IndexerOverrides: ov,
				}
			}
		}
		if _, exists := dm.devices["admin"]; exists {
			delete(dm.devices, "admin")
			dm.saveLocked()
			logger.Info("Removed legacy admin from devices (admin is in config)")
		}
		return nil
	}

	var devices map[string]*Device
	found, err := dm.manager.Get("devices", &devices)
	if err != nil {
		return err
	}
	if !found {
		var users map[string]*Device
		if found, err := dm.manager.Get("users", &users); found && err == nil {
			devices = users
			dm.manager.Set("devices", devices)
			logger.Info("Migrated users to devices in state.json")
		}
	}
	if devices != nil {
		dm.devices = make(map[string]*Device)
		for k, d := range devices {
			if d == nil {
				continue
			}
			dm.devices[k] = &Device{
				Username:         d.Username,
				Token:            d.Token,
				IndexerOverrides: d.IndexerOverrides,
			}
			if dm.devices[k].IndexerOverrides == nil {
				dm.devices[k].IndexerOverrides = make(map[string]config.IndexerSearchConfig)
			}
		}
		if _, exists := dm.devices["admin"]; exists {
			delete(dm.devices, "admin")
			dm.saveLocked()
			logger.Info("Removed legacy admin from devices (admin is in config)")
		}
	} else {
		dm.devices = make(map[string]*Device)
	}
	return nil
}

func (dm *DeviceManager) saveLocked() error {
	if dm.cfg != nil {
		dm.cfg.Devices = make(map[string]*config.DeviceEntry)
		for k, d := range dm.devices {
			ov := d.IndexerOverrides
			if ov == nil {
				ov = make(map[string]config.IndexerSearchConfig)
			}
			dm.cfg.Devices[k] = &config.DeviceEntry{
				Username:         d.Username,
				Token:            d.Token,
				IndexerOverrides: ov,
			}
		}
		if dm.saveFn != nil {
			return dm.saveFn()
		}
		return nil
	}
	return dm.manager.Set("devices", dm.devices)
}

func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

func (dm *DeviceManager) Authenticate(loginUsername, password, adminUsername, adminPasswordHash, adminToken string) (*Device, error) {
	if adminUsername == "" {
		adminUsername = "admin"
	}

	if loginUsername == adminUsername {
		if adminPasswordHash == "" || adminToken == "" {
			return nil, fmt.Errorf("invalid credentials")
		}
		passwordHash := HashPassword(password)
		if passwordHash != adminPasswordHash {
			return nil, fmt.Errorf("invalid credentials")
		}
		return &Device{
			Username:         adminUsername,
			Token:            adminToken,
			IndexerOverrides: nil,
		}, nil
	}

	return nil, fmt.Errorf("invalid credentials")
}

func (dm *DeviceManager) AuthenticateToken(token string, adminUsername, adminToken string) (*Device, error) {
	if adminUsername == "" {
		adminUsername = "admin"
	}

	if adminToken != "" && token == adminToken {
		return &Device{
			Username:         adminUsername,
			Token:            adminToken,
			IndexerOverrides: nil,
		}, nil
	}

	dm.mu.RLock()
	defer dm.mu.RUnlock()
	for _, device := range dm.devices {
		if device.Token == token {
			return device, nil
		}
	}

	return nil, fmt.Errorf("invalid token")
}

func (dm *DeviceManager) GetDevice(username string, adminUsername string) (*Device, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if adminUsername == "" {
		adminUsername = "admin"
	}
	if username == adminUsername {
		return nil, fmt.Errorf("admin is not a regular device")
	}

	device, exists := dm.devices[username]
	if !exists {
		return nil, fmt.Errorf("device not found")
	}

	return device, nil
}

func (dm *DeviceManager) GetUser(username string, adminUsername string) (*Device, error) {
	return dm.GetDevice(username, adminUsername)
}

func (dm *DeviceManager) GetAllDevices() []Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]Device, 0, len(dm.devices))
	for _, device := range dm.devices {

		if device.Username == "admin" {
			continue
		}
		devices = append(devices, Device{
			Username:         device.Username,
			Token:            device.Token,
			IndexerOverrides: device.IndexerOverrides,
		})
	}

	return devices
}

func (dm *DeviceManager) GetAllUsers() []Device {
	return dm.GetAllDevices()
}

func (dm *DeviceManager) CreateDevice(username, password string, adminUsername string) (*Device, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if adminUsername == "" {
		adminUsername = "admin"
	}
	if username == adminUsername {
		return nil, fmt.Errorf("cannot create admin device via this method")
	}

	if _, exists := dm.devices[username]; exists {
		return nil, fmt.Errorf("device already exists")
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	device := &Device{
		Username:         username,
		Token:            token,
		IndexerOverrides: make(map[string]config.IndexerSearchConfig),
	}

	dm.devices[username] = device

	if err := dm.saveLocked(); err != nil {
		delete(dm.devices, username)
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	logger.Info("Created device", "username", username)
	return device, nil
}

func (dm *DeviceManager) CreateUser(username, password string, adminUsername string) (*Device, error) {
	return dm.CreateDevice(username, password, adminUsername)
}

func (dm *DeviceManager) UpdateUser(username, newPassword string, adminUsername string) error {
	if adminUsername == "" {
		adminUsername = "admin"
	}
	if username == adminUsername {
		return fmt.Errorf("admin password is managed via config; use dashboard to change")
	}
	return fmt.Errorf("only admin password can be updated")
}

func (dm *DeviceManager) RegenerateToken(username string) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device, exists := dm.devices[username]
	if !exists {
		return "", fmt.Errorf("device not found")
	}

	token, err := GenerateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	device.Token = token

	if err := dm.saveLocked(); err != nil {
		return "", fmt.Errorf("failed to save device: %w", err)
	}

	logger.Info("Regenerated token for device", "username", username)
	return token, nil
}

func (dm *DeviceManager) DeleteDevice(username string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.devices[username]; !exists {
		return fmt.Errorf("device not found")
	}

	delete(dm.devices, username)

	if err := dm.saveLocked(); err != nil {
		return fmt.Errorf("failed to save device: %w", err)
	}

	logger.Info("Deleted device", "username", username)
	return nil
}

func (dm *DeviceManager) DeleteUser(username string) error {
	return dm.DeleteDevice(username)
}

func (dm *DeviceManager) UpdateDeviceIndexerOverrides(username string, overrides map[string]config.IndexerSearchConfig) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device, exists := dm.devices[username]
	if !exists {
		return fmt.Errorf("device not found")
	}

	if overrides == nil {
		device.IndexerOverrides = make(map[string]config.IndexerSearchConfig)
	} else {
		device.IndexerOverrides = overrides
	}

	if err := dm.saveLocked(); err != nil {
		return fmt.Errorf("failed to save device indexer overrides: %w", err)
	}
	return nil
}


