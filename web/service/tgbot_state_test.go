package service

import (
	"math"
	"strconv"
	"sync"
	"testing"
)

func TestTgbotUserStateStoreConcurrentAccess(t *testing.T) {
	const workers = 32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chatID := int64(i)
			state := "state-" + strconv.Itoa(i)
			setUserState(chatID, state)
			if got, ok := getUserState(chatID); !ok || got != state {
				t.Errorf("getUserState(%d) = %q, %v", chatID, got, ok)
			}
			deleteUserState(chatID)
			if _, ok := getUserState(chatID); ok {
				t.Errorf("state for chat %d was not deleted", chatID)
			}
		}(i)
	}

	wg.Wait()
}

func TestClientAddStateIsIsolatedByChat(t *testing.T) {
	clientStateMutex.Lock()
	defer clientStateMutex.Unlock()

	resetClientAddGlobals()
	receiver_inbound_ID = 1
	client_Email = "first@example.test"
	saveClientAddState(101)

	resetClientAddGlobals()
	receiver_inbound_ID = 2
	client_Email = "second@example.test"
	saveClientAddState(202)

	loadClientAddState(101)
	if receiver_inbound_ID != 1 || client_Email != "first@example.test" {
		t.Fatalf("chat 101 state = inbound %d email %q", receiver_inbound_ID, client_Email)
	}

	loadClientAddState(202)
	if receiver_inbound_ID != 2 || client_Email != "second@example.test" {
		t.Fatalf("chat 202 state = inbound %d email %q", receiver_inbound_ID, client_Email)
	}

	resetClientAddGlobals()
	saveClientAddState(101)
	saveClientAddState(202)
}

func TestTgbotNumericHelpersClampOverflow(t *testing.T) {
	if got := saturatingAddUint64(math.MaxUint64, 1); got != math.MaxUint64 {
		t.Fatalf("saturatingAddUint64 overflow = %d, want %d", got, uint64(math.MaxUint64))
	}
	if got := telegramRequestID(int(math.MaxInt32) + 1); got != math.MaxInt32 {
		t.Fatalf("telegramRequestID overflow = %d, want %d", got, int32(math.MaxInt32))
	}
}
