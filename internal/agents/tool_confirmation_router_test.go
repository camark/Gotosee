package agents

import (
	"testing"

	"github.com/aaif-goose/gogo/internal/permission"
)

func TestToolConfirmationRouter_RegisterAndDeliver(t *testing.T) {
	router := NewToolConfirmationRouter()

	ch := router.Register("req-123")

	confirmation := PermissionConfirmation{
		Permission: permission.AllowOnce,
	}

	delivered := router.Deliver("req-123", confirmation)
	if !delivered {
		t.Fatal("Failed to deliver confirmation")
	}

	received := <-ch
	if received.Permission != permission.AllowOnce {
		t.Errorf("Expected AllowOnce, got %v", received.Permission)
	}
}

func TestToolConfirmationRouter_DeliverToNonExistent(t *testing.T) {
	router := NewToolConfirmationRouter()

	delivered := router.Deliver("non-existent", PermissionConfirmation{})
	if delivered {
		t.Error("Should not deliver to non-existent request")
	}
}

func TestToolConfirmationRouter_MultipleRegistrations(t *testing.T) {
	router := NewToolConfirmationRouter()

	ch1 := router.Register("req-1")
	ch2 := router.Register("req-2")
	ch3 := router.Register("req-3")

	// Deliver to all
	router.Deliver("req-1", PermissionConfirmation{Permission: permission.AllowOnce})
	router.Deliver("req-2", PermissionConfirmation{Permission: permission.AlwaysAllow})
	router.Deliver("req-3", PermissionConfirmation{Permission: permission.DenyOnce})

	// Verify each channel received correct value
	r1 := <-ch1
	if r1.Permission != permission.AllowOnce {
		t.Errorf("req-1: expected AllowOnce, got %v", r1.Permission)
	}

	r2 := <-ch2
	if r2.Permission != permission.AlwaysAllow {
		t.Errorf("req-2: expected AlwaysAllow, got %v", r2.Permission)
	}

	r3 := <-ch3
	if r3.Permission != permission.DenyOnce {
		t.Errorf("req-3: expected DenyOnce, got %v", r3.Permission)
	}
}

func TestToolConfirmationRouter_ChannelClosedAfterDelivery(t *testing.T) {
	router := NewToolConfirmationRouter()

	ch := router.Register("req-test")
	router.Deliver("req-test", PermissionConfirmation{Permission: permission.AllowOnce})

	// Receive the value
	<-ch

	// Channel should be closed now
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed after delivery")
		}
	default:
		t.Error("Channel should be detectable as closed")
	}
}
