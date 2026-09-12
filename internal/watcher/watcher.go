package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event はファイルの変更イベントを表します。
type Event struct {
	Path    string
	Content string
	ModTime time.Time
	Err     error
}

// Option は Watcher の設定を変更する関数型です。
type Option func(*Watcher)

// WithDebounce はデバウンス時間を設定します（デフォルト: 100ms）。
func WithDebounce(d time.Duration) Option {
	return func(w *Watcher) {
		w.debounce = d
	}
}

// Watcher は対象ファイルの変更を監視し、イベントを送信します。
type Watcher struct {
	filePath    string
	absPath     string
	parentDir   string
	baseName    string
	fsWatcher   *fsnotify.Watcher
	debounce    time.Duration
	events      chan Event
	done        chan struct{}
	closeOnce   sync.Once
	mu          sync.Mutex
	lastContent string
}

// New は新しい Watcher を生成して対象ファイルの親ディレクトリの監視を開始します。
func New(filePath string, opts ...Option) (*Watcher, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	parentDir := filepath.Dir(absPath)
	baseName := filepath.Base(absPath)

	initialContent, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial file: %w", err)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		filePath:    filePath,
		absPath:     absPath,
		parentDir:   parentDir,
		baseName:    baseName,
		fsWatcher:   fsWatcher,
		debounce:    100 * time.Millisecond,
		events:      make(chan Event, 10),
		done:        make(chan struct{}),
		lastContent: string(initialContent),
	}

	for _, opt := range opts {
		opt(w)
	}

	// 親ディレクトリを監視する（各種エディタのアトミックセーブによる Rename/Create に対応するため）
	if err := fsWatcher.Add(parentDir); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("failed to watch directory %s: %w", parentDir, err)
	}

	go w.watchLoop()

	return w, nil
}

// Events はファイル変更イベントを受信するチャネルを返します。
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Close は監視を停止しリソースを解放します。
func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.fsWatcher.Close()
	})
	return err
}

func (w *Watcher) isTargetEvent(event fsnotify.Event) bool {
	eventBase := filepath.Base(event.Name)
	if eventBase != w.baseName {
		return false
	}

	// Write, Create, Rename などの変更イベントを対象とする
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		return true
	}
	return false
}

func (w *Watcher) watchLoop() {
	defer close(w.events)

	var (
		timer   *time.Timer
		timerCh <-chan time.Time
	)

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			if w.isTargetEvent(event) {
				if timer != nil {
					timer.Stop()
				}
				timer = time.NewTimer(w.debounce)
				timerCh = timer.C
			}

		case <-timerCh:
			timerCh = nil
			w.triggerReload()

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.events <- Event{Err: err}:
			case <-w.done:
				return
			}
		}
	}
}

func (w *Watcher) triggerReload() {
	// エディタのアトミック保存時の瞬間的なファイル不在や0バイト状態に対応するため、数回リトライする
	var (
		content []byte
		err     error
		info    os.FileInfo
	)

	for i := 0; i < 4; i++ {
		content, err = os.ReadFile(w.absPath)
		if err == nil {
			info, err = os.Stat(w.absPath)
			if err == nil && (len(content) > 0 || info.Size() == 0) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	if err != nil {
		select {
		case w.events <- Event{Path: w.absPath, Err: err}:
		case <-w.done:
		}
		return
	}

	w.mu.Lock()
	newContent := string(content)
	if newContent == w.lastContent {
		w.mu.Unlock()
		return
	}
	w.lastContent = newContent
	w.mu.Unlock()

	var modTime time.Time
	if info != nil {
		modTime = info.ModTime()
	} else {
		modTime = time.Now()
	}

	select {
	case w.events <- Event{
		Path:    w.absPath,
		Content: newContent,
		ModTime: modTime,
	}:
	case <-w.done:
	}
}
