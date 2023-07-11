// mutex パッケージは http のアクセス権制御を提供します.
package mutex

import (
	"context"
	"sync"
	"time"

	"github.com/17e10/go-notifyb"
)

const Cancel notifyb.Notify = "cancel"

// Mutex は http のアクセス権を制御します.
type Mutex struct {
	d    time.Duration // アクセス間隔
	next time.Time     // 次回のアクセス可能時刻
	mu   sync.Mutex    // アクセス競合を制御する Mutex
}

// New は新しい Mutex を作成します.
func New(d time.Duration) *Mutex {
	return &Mutex{d: d}
}

// Lock は http のアクセス権を取得します.
// context.Context がキャンセルされたとき Cancel が返されます.
func (m *Mutex) Lock(ctx context.Context) error {
	m.mu.Lock()
	now := time.Now()
	select {
	case <-ctx.Done():
		return Cancel
	case <-time.After(m.next.Sub(now)):
		return nil
	}
}

// Unlock は http のアクセス権を解放します.
func (m *Mutex) Unlock() {
	m.next = time.Now().Add(m.d)
	m.mu.Unlock()
}
