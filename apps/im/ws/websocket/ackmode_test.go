package websocket

import "testing"

func TestResolveAckModes(t *testing.T) {
	cases := []struct {
		name        string
		ws          string
		obs         string
		enabled     bool
		wantAck     AckType
		wantObserve bool
		wantErr     bool
	}{
		{"default NoAck", "", "", true, NoAck, false, false},
		{"NoAck aligned", "NoAck", "NoAck", true, NoAck, false, false},
		{"OnlyAck no fake ack events", "OnlyAck", "OnlyAck", true, OnlyAck, false, false},
		{"RigorAck observe", "RigorAck", "RigorAck", true, RigorAck, true, false},
		{"RigorAck but observation off", "RigorAck", "RigorAck", false, RigorAck, false, false},
		{"mismatch refuses", "RigorAck", "NoAck", true, 0, false, true},
		{"invalid ws", "FakeAck", "NoAck", true, 0, false, true},
		{"invalid obs", "NoAck", "FakeAck", true, 0, false, true},
	}

	for _, tc := range cases {
		wsAck, obsAck, ackObserve, err := ResolveAckModes(tc.ws, tc.obs, tc.enabled)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error", tc.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error %v", tc.name, err)
		}
		if wsAck != tc.wantAck {
			t.Fatalf("%s: wsAck=%v want %v", tc.name, wsAck, tc.wantAck)
		}
		if obsAck != tc.wantAck.ToString() {
			t.Fatalf("%s: obsAck=%s", tc.name, obsAck)
		}
		if ackObserve != tc.wantObserve {
			t.Fatalf("%s: ackObserve=%v want %v", tc.name, ackObserve, tc.wantObserve)
		}
	}
}

func TestParseAckType(t *testing.T) {
	if ack, err := ParseAckType("RigorAck"); err != nil || ack != RigorAck {
		t.Fatalf("got %v %v", ack, err)
	}
	if ack, err := ParseAckType(""); err != nil || ack != NoAck {
		t.Fatalf("empty must be NoAck, got %v %v", ack, err)
	}
	if _, err := ParseAckType("nope"); err == nil {
		t.Fatal("invalid must error")
	}
}
