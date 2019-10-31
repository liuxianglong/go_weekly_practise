package io

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RollingManager interface {
	Initialize(rfw *RollingFileWriter) error
	ShouldRoll() (isShould bool, targetName string)
}

func NewRollingFileWriter(filePath string, rmg RollingManager) (rw *RollingFileWriter, err error) {
	rw = &RollingFileWriter{
		path: filePath,
		rmg:  rmg,
	}
	err = rw.initialize()
	return
}

// RotateFileWriter is file writer with rotation
type RollingFileWriter struct {
	rwMutex sync.RWMutex
	file    *os.File
	path    string
	size    int64
	rv      int64
	rmg     RollingManager
}

func (r *RollingFileWriter) initialize() (err error) {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()

	logDir := filepath.Dir(r.path)
	if logDir != "." {
		err = os.MkdirAll(logDir, os.FileMode(0755))
		if err != nil {
			return
		}
	}
	err = r.openFile()
	if err != nil {
		return
	}

	var fi os.FileInfo
	fi, err = r.file.Stat()
	if err != nil {
		return
	}

	r.size = fi.Size()
	err = r.rmg.Initialize(r)
	return
}

func (r *RollingFileWriter) Write(b []byte) (n int, err error) {
	r.checkRolling()

	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()

	n, err = r.file.Write(b)
	atomic.AddInt64(&r.size, int64(n))
	return
}

func (r *RollingFileWriter) WriteString(s string) (n int, err error) {
	r.checkRolling()

	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()

	n, err = r.file.WriteString(s)
	atomic.AddInt64(&r.size, int64(n))
	return
}

func (r *RollingFileWriter) checkRolling() {
	ver := r.rv
	if shouldRoll, targetName := r.rmg.ShouldRoll(); shouldRoll {
		r.rwMutex.Lock()
		defer r.rwMutex.Unlock()

		if ver == r.rv { // check roll version
			r.roll(targetName)
			r.rv++
		}
	}
}

func (r *RollingFileWriter) roll(targetName string) (err error) {
	err = r.file.Close()
	if err != nil {
		return
	}

	newPath := filepath.Dir(r.path) + string(filepath.Separator) + targetName
	os.Rename(r.path, newPath)

	err = r.openFile()
	if err != nil {
		return
	}

	atomic.StoreInt64(&r.size, int64(0))
	return
}

func (r *RollingFileWriter) Roll(targetName string) (err error) {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()

	err = r.roll(targetName)
	return
}

func (r *RollingFileWriter) openFile() (err error) {
	r.file, err = os.OpenFile(r.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	return
}

func (r *RollingFileWriter) Size() int64 {
	return r.size
}

func (r *RollingFileWriter) FileInfo() (info os.FileInfo, err error) {
	if r.file != nil {
		info, err = r.file.Stat()
		return
	}

	if r.path != "" {
		info, err = os.Stat(r.path)
	}

	return nil, os.ErrNotExist
}

func (r *RollingFileWriter) Close() error {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()

	return r.file.Close()
}

type dailyRollingManager struct {
	fileName     string
	fileExt      string
	rollingPoint time.Time
	rfw          *RollingFileWriter
}

func beginningOfTheDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (d *dailyRollingManager) Initialize(rfw *RollingFileWriter) error {
	fileInfo, err := rfw.FileInfo()
	if err != nil {
		return err
	}

	d.rfw = rfw
	d.fileExt = filepath.Ext(fileInfo.Name())
	d.fileName = strings.TrimSuffix(fileInfo.Name(), d.fileExt)
	if d.rfw.Size() > 0 {
		d.rollingPoint = beginningOfTheDate(fileInfo.ModTime().AddDate(0, 0, 1))
	} else {
		d.rollingPoint = beginningOfTheDate(time.Now().AddDate(0, 0, 1))
	}

	return nil
}

func (d *dailyRollingManager) ShouldRoll() (isShould bool, targetName string) {
	now := time.Now()
	if now.Before(d.rollingPoint) {
		return
	}

	if d.rfw.Size() == 0 {
		d.rollingPoint = beginningOfTheDate(now.AddDate(0, 0, 1))
		return
	}

	isShould = true
	targetName = d.fileName + "." + d.rollingPoint.AddDate(0, 0, -1).Format("20060102") + d.fileExt
	d.rollingPoint = beginningOfTheDate(now.AddDate(0, 0, 1))
	return
}

type timePatternRollingManager struct {
	fileName  string
	fileExt   string
	pattern   string
	timestamp string
	rfw       *RollingFileWriter
}

var errTimePatternNotFound = errors.New("time pattern not found")

func (t *timePatternRollingManager) Initialize(rfw *RollingFileWriter) error {
	if t.pattern == "" {
		return errTimePatternNotFound
	}

	fileInfo, err := rfw.FileInfo()
	if err != nil {
		return err
	}

	t.rfw = rfw
	t.fileExt = filepath.Ext(fileInfo.Name())
	t.fileName = strings.TrimSuffix(fileInfo.Name(), t.fileExt)
	if t.rfw.Size() > 0 {
		t.timestamp = fileInfo.ModTime().Format(t.pattern)
	} else {
		t.timestamp = time.Now().Format(t.pattern)
	}

	return nil
}

func (t *timePatternRollingManager) ShouldRoll() (isShould bool, targetName string) {
	currentTimestamp := time.Now().Format(t.pattern)
	if currentTimestamp == t.timestamp {
		return
	}

	if t.rfw.Size() == 0 {
		t.timestamp = currentTimestamp
		return
	}

	isShould = true
	targetName = t.fileName + "." + t.timestamp + t.fileExt
	t.timestamp = currentTimestamp
	return
}

func NewTimePatternRollingManager(pattern string) RollingManager {
	return &timePatternRollingManager{
		pattern: pattern,
	}
}

func NewDailyRollingManager() RollingManager {
	return &dailyRollingManager{}
}
