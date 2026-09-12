package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher_FileModification(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.md")

	initialContent := "# Initial Content\n"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	w, err := New(testFile, WithDebounce(20*time.Millisecond))
	if err != nil {
		t.Fatalf("New watcher failed: %v", err)
	}
	defer w.Close()

	updatedContent := "# Updated Content\n\nSome new paragraph."
	if err := os.WriteFile(testFile, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	select {
	case ev, ok := <-w.Events():
		if !ok {
			t.Fatalf("Events channel closed unexpectedly")
		}
		if ev.Err != nil {
			t.Fatalf("Received error event: %v", ev.Err)
		}
		if ev.Content != updatedContent {
			t.Errorf("Expected content %q, got %q", updatedContent, ev.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for file change event")
	}
}

func TestWatcher_AtomicSave(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.md")

	initialContent := "# Initial\n"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	w, err := New(testFile, WithDebounce(20*time.Millisecond))
	if err != nil {
		t.Fatalf("New watcher failed: %v", err)
	}
	defer w.Close()

	// エディタによるアトミック保存のシミュレーション:
	// 一時ファイルに書いてから、対象ファイルにリネーム（上書き）する
	tempSaveFile := filepath.Join(tempDir, "test.md.tmp")
	atomicContent := "# Atomic Save Content\n"
	if err := os.WriteFile(tempSaveFile, []byte(atomicContent), 0644); err != nil {
		t.Fatalf("Failed to write temp save file: %v", err)
	}
	if err := os.Rename(tempSaveFile, testFile); err != nil {
		t.Fatalf("Failed to rename temp file: %v", err)
	}

	select {
	case ev, ok := <-w.Events():
		if !ok {
			t.Fatalf("Events channel closed unexpectedly")
		}
		if ev.Err != nil {
			t.Fatalf("Received error event: %v", ev.Err)
		}
		if ev.Content != atomicContent {
			t.Errorf("Expected content %q, got %q", atomicContent, ev.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for atomic save event")
	}
}

func TestWatcher_NoEventOnSameContent(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.md")

	content := "# Static Content\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	w, err := New(testFile, WithDebounce(20*time.Millisecond))
	if err != nil {
		t.Fatalf("New watcher failed: %v", err)
	}
	defer w.Close()

	// 同じ内容を書き込む
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write same content: %v", err)
	}

	select {
	case ev := <-w.Events():
		t.Fatalf("Unexpected event for unchanged content: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// 期待通りイベントが発火しない
	}
}

func TestWatcher_Close(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.md")

	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	w, err := New(testFile)
	if err != nil {
		t.Fatalf("New watcher failed: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// チャネルがクローズされること
	select {
	case _, ok := <-w.Events():
		if ok {
			t.Errorf("Expected closed events channel")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timeout waiting for events channel to close")
	}
}
