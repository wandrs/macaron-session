// Copyright 2013 Beego Authors
// Copyright 2014 The Macaron Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package session

import (
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/unknwon/com"
)

// FileStore represents a file session store implementation.
type FileStore struct {
	p    *FileProvider
	sid  string
	lock sync.RWMutex
	data map[interface{}]interface{}
}

// NewFileStore creates and returns a file session store.
func NewFileStore(p *FileProvider, sid string, kv map[interface{}]interface{}) *FileStore {
	return &FileStore{
		p:    p,
		sid:  sid,
		data: kv,
	}
}

// Set sets value to given key in session.
func (s *FileStore) Set(key, val interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[key] = val
	return nil
}

// Get gets value by given key in session.
func (s *FileStore) Get(key interface{}) interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.data[key]
}

// Delete delete a key from session.
func (s *FileStore) Delete(key interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.data, key)
	return nil
}

// ID returns current session ID.
func (s *FileStore) ID() string {
	return s.sid
}

// Release releases resource and save data to provider.
func (s *FileStore) Release() error {
	s.p.lock.Lock()
	defer s.p.lock.Unlock()

	// Skip encoding if the data is empty
	if len(s.data) == 0 {
		return nil
	}

	data, err := EncodeGob(s.data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.p.filepath(s.sid), data, 0600)
}

// Flush deletes all session data.
func (s *FileStore) Flush() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data = make(map[interface{}]interface{})
	return nil
}

/* ================================== FileHubStore ======================================== */

type FileHubStore struct {
	p               *FileProvider
	uid, sessPrefix string
	lock            sync.RWMutex
	cleanup         map[string]struct{}
	data            UserSessionHub
}

var _ HubStore = (*FileHubStore)(nil)

func NewFileHubStore(p *FileProvider, uid string, data UserSessionHub) (*FileHubStore, error) {
	return &FileHubStore{
		p:       p,
		uid:     uid,
		data:    data,
		cleanup: make(map[string]struct{}),
	}, nil
}

func getHubStoreKey(uid string) string {
	return fmt.Sprintf("sessions.user.%v", uid)
}

func (h *FileHubStore) Add(sessionKey string, si SessInfo) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.data.SessionStore[sessionKey] = si
	return nil
}

func (h *FileHubStore) Remove(sessionKey string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	delete(h.data.SessionStore, sessionKey)
	h.cleanup[sessionKey] = struct{}{}
	return nil
}

func (h *FileHubStore) RemoveAll() error {
	h.lock.Lock()
	defer h.lock.Unlock()

	for k := range h.data.SessionStore {
		h.cleanup[k] = struct{}{}
	}
	h.data.SessionStore = make(map[string]SessInfo)
	return nil
}

func (h *FileHubStore) RemoveExcept(sessionKey string) error {
	_ = h.addForCleanUpExcept(sessionKey)

	backupVal := h.data.SessionStore[sessionKey]
	h.data.SessionStore = map[string]SessInfo{}
	_ = h.Add(sessionKey, backupVal)

	return nil
}

func (h *FileHubStore) addForCleanUpExcept(sessionKey string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	for k := range h.data.SessionStore {
		h.cleanup[k] = struct{}{}
	}
	delete(h.cleanup, sessionKey)

	return nil
}

func (h *FileHubStore) flushExpired() error {
	backupData := make(map[string]SessInfo)
	for k, si := range h.data.SessionStore {
		backupData[k] = si
	}
	currentTime := time.Now()
	for k, si := range backupData {
		if si.Exp.Before(currentTime) {
			h.cleanup[k] = struct{}{}
			delete(h.data.SessionStore, k)
		}
	}

	return nil
}

func (h *FileHubStore) ReleaseHubData() error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if len(h.data.SessionStore) == 0 {
		return nil
	}
	_ = h.flushExpired()

	data, err := EncodedUserSessionHub(h.data)
	if err != nil {
		return err
	}

	_ = h.executeCleanup()

	hubKey := getHubStoreKey(h.uid)

	return os.WriteFile(h.p.filepath(hubKey), data, 0600)
}

func (h *FileHubStore) executeCleanup() error {
	for sid, _ := range h.cleanup {
		if err := os.Remove(h.p.filepath(sid)); err != nil && !os.IsNotExist(err) {
			slog.Error("failed to cleanup session: %v", err)
		}
	}

	return nil
}

func (h *FileHubStore) List() []SessInfo {
	sessions := make([]SessInfo, 0)
	for _, si := range h.data.SessionStore {
		if si.SessionID == "" {
			continue
		}
		sessions = append(sessions, si)
	}

	return sessions
}

/* ================================== FileProvider ======================================== */

// FileProvider represents a file session provider implementation.
type FileProvider struct {
	lock        sync.RWMutex
	maxlifetime int64
	rootPath    string
}

var _ Provider = (*FileProvider)(nil)

// Init initializes file session provider with given root path.
func (p *FileProvider) Init(maxlifetime int64, rootPath string) error {
	p.lock.Lock()
	mlt, err := time.ParseDuration(fmt.Sprintf("%ds", maxlifetime))
	if err != nil {
		return err
	}
	p.maxlifetime = int64(mlt)
	p.rootPath = rootPath
	p.lock.Unlock()
	return nil
}

func (p *FileProvider) filepath(sid string) string {
	return path.Join(p.rootPath, string(sid[0]), string(sid[1]), sid)
}

// Read returns raw session store by session ID.
func (p *FileProvider) Read(sid string) (_ RawStore, err error) {
	filename := p.filepath(sid)
	if err = os.MkdirAll(path.Dir(filename), 0700); err != nil {
		return nil, err
	}
	p.lock.RLock()
	defer p.lock.RUnlock()

	var f *os.File
	expired := true
	if com.IsFile(filename) {
		modTime, err := com.FileMTime(filename)
		if err != nil {
			return nil, err
		}
		expired = (modTime + p.maxlifetime) < time.Now().Unix()
	}
	if !expired {
		f, err = os.OpenFile(filename, os.O_RDONLY, 0600)
	} else {
		f, err = os.Create(filename)
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err = os.Chtimes(filename, time.Now(), time.Now()); err != nil {
		return nil, err
	}

	var kv map[interface{}]interface{}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = DecodeGob(data)
		if err != nil {
			return nil, err
		}
	}
	return NewFileStore(p, sid, kv), nil
}

// Exist returns true if session with given ID exists.
func (p *FileProvider) Exist(sid string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return com.IsFile(p.filepath(sid))
}

// Destory deletes a session by session ID.
func (p *FileProvider) Destory(sid string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	return os.Remove(p.filepath(sid))
}

func (p *FileProvider) regenerate(oldsid, sid string) (err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	filename := p.filepath(sid)
	if com.IsExist(filename) {
		return fmt.Errorf("new sid '%s' already exists", sid)
	}

	oldname := p.filepath(oldsid)
	if !com.IsFile(oldname) {
		data, err := EncodeGob(make(map[interface{}]interface{}))
		if err != nil {
			return err
		}
		if err = os.MkdirAll(path.Dir(oldname), 0700); err != nil {
			return err
		}
		if err = ioutil.WriteFile(oldname, data, 0600); err != nil {
			return err
		}
	}

	if err = os.MkdirAll(path.Dir(filename), 0700); err != nil {
		return err
	}
	if err = os.Rename(oldname, filename); err != nil {
		return err
	}
	return nil
}

// Regenerate regenerates a session store from old session ID to new one.
func (p *FileProvider) Regenerate(oldsid, sid string) (_ RawStore, err error) {
	if err := p.regenerate(oldsid, sid); err != nil {
		return nil, err
	}

	return p.Read(sid)
}

// Count counts and returns number of sessions.
func (p *FileProvider) Count() int {
	count := 0
	if err := filepath.Walk(p.rootPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			count++
		}
		return nil
	}); err != nil {
		log.Printf("error counting session files: %v", err)
		return 0
	}
	return count
}

// GC calls GC to clean expired sessions.
func (p *FileProvider) GC() {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if !com.IsExist(p.rootPath) {
		return
	}

	if err := filepath.Walk(p.rootPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.IsDir() &&
			(fi.ModTime().Unix()+p.maxlifetime) < time.Now().Unix() {
			return os.Remove(path)
		}
		return nil
	}); err != nil {
		log.Printf("error garbage collecting session files: %v", err)
	}
}

// ReadSessionHubStore returns the RawStore which manipulates the session hub data of a user
func (p *FileProvider) ReadSessionHubStore(uid string) (HubStore, error) {
	hubKey := getHubStoreKey(uid)
	filename := p.filepath(hubKey)

	_, err := os.Stat(p.filepath(hubKey))
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path.Dir(filename), 0700)
			if err != nil {
				return nil, fmt.Errorf("failed to create dirs: %v", err)
			}
			if _, err = os.Create(filename); err != nil {
				return nil, fmt.Errorf("failed to create hub store file: %v", err)
			}
			data, err := NewEmptyUserSessionData(uid)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(filename, data, os.ModePerm); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("failed to create hub store file: %v", err)
		}
	}

	result, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var hubData UserSessionHub
	if len(result) > 0 {
		hubData, err = DecodeUserSessionHub(result)
		if err != nil {
			return nil, err
		}
	}

	return NewFileHubStore(p, uid, hubData)
}

func (h *FileHubStore) IsExistsSession(sid string) bool {
	h.lock.Lock()
	defer h.lock.Unlock()

	_, found := h.data.SessionStore[sid]

	return found
}

// FlushNonCompatibleData deletes the older versions of UserSessionHub data
func (p *FileProvider) FlushNonCompatibleData() error {
	return nil
}

// SessionDuration returns the duration set for the session
func (p *FileProvider) SessionDuration() time.Duration {
	return time.Duration(p.maxlifetime)
}

func init() {
	Register("file", &FileProvider{})
}
