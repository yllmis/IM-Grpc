package logic

import (
	"context"
	"testing"

	"github.com/IM_System/apps/im/immodels"
	"github.com/IM_System/apps/operations/rpc/internal/faultinject"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/apps/operations/rpc/operationsmodels"
	"github.com/IM_System/pkg/constants"
	"github.com/IM_System/pkg/observation"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- fakes ----

type fakeEventModel struct {
	events []*observation.MessageEvent
	gaps   uint64
	gapOK  bool
	err    error
}

func (f *fakeEventModel) FindByMessageID(_ context.Context, _ operationsmodels.EventQueryFilter) ([]*observation.MessageEvent, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.events, nil
}
func (f *fakeEventModel) FindByReceiver(_ context.Context, q operationsmodels.EventQueryFilter) ([]*observation.MessageEvent, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*observation.MessageEvent
	for _, ev := range f.events {
		if q.ReceiverID == "" || ev.ReceiverID == q.ReceiverID {
			out = append(out, ev)
		}
	}
	return out, nil
}
func (f *fakeEventModel) FindBySender(_ context.Context, _ operationsmodels.EventQueryFilter) ([]*observation.MessageEvent, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.events, nil
}
func (f *fakeEventModel) HasAnyByMessageID(_ context.Context, _ string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return len(f.events) > 0, nil
}
func (f *fakeEventModel) CountGaps(_ context.Context, _, _ int64) (uint64, bool, error) {
	if f.err != nil {
		return 0, false, f.err
	}
	return f.gaps, f.gapOK, nil
}
func (f *fakeEventModel) FindOneByID(_ context.Context, _ string) (*observation.MessageEvent, error) {
	return nil, operationsmodels.ErrNotFound
}

type fakeChatLogModel struct {
	immodels.ChatLogModel
	log *immodels.ChatLog
	err error
}

func (f *fakeChatLogModel) FindOne(_ context.Context, _ string) (*immodels.ChatLog, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.log, nil
}

func codeOf(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return status.Code(err)
}

// ---- cursor ----

func TestCursorRoundTrip(t *testing.T) {
	c := EncodeCursor(123, "abc")
	at, id, err := DecodeCursor(c)
	if err != nil || at != 123 || id != "abc" {
		t.Fatalf("roundtrip: %d %q %v", at, id, err)
	}
	if _, _, err := DecodeCursor("!!!bad"); codeOf(err) != codes.InvalidArgument {
		t.Fatal("bad cursor must be INVALID_ARGUMENT")
	}
}

// ---- GetMessageRecord ----

func TestGetMessageRecord_InvalidID(t *testing.T) {
	l := &GetMessageRecordLogic{ctx: context.Background(), svcCtx: &svc.ServiceContext{}}
	_, err := l.GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: ""})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("empty id must INVALID_ARGUMENT, got %v", err)
	}
	_, err = l.GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: "bad"})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("bad id must INVALID_ARGUMENT, got %v", err)
	}
}

func TestGetMessageRecord_NotFoundIsOK(t *testing.T) {
	svcCtx := &svc.ServiceContext{
		ChatLogModel: &fakeChatLogModel{err: immodels.ErrNotFound},
		EventModel:   &fakeEventModel{},
	}
	l := NewGetMessageRecordLogic(context.Background(), svcCtx)

	resp, err := l.GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: "665f1c0000000000000000aa"})
	if err != nil {
		t.Fatalf("not found must be OK empty result, got %v", err)
	}
	if resp.Found {
		t.Fatal("found must be false")
	}
	if resp.Note == "" {
		t.Fatal("note must explain query-succeeded-no-record")
	}
}

func TestFaultInjectionMessageScenariosAreReadOnly(t *testing.T) {
	id := "665f1c0000000000000000aa"
	for _, tc := range []struct {
		name     string
		scenario faultinject.Scenario
		wantCode codes.Code
		wantNote string
	}{
		{name: "missing", scenario: faultinject.MessageMissing, wantNote: "query-succeeded-no-record"},
		{name: "timeout", scenario: faultinject.QueryTimeout, wantCode: codes.DeadlineExceeded},
		{name: "permission", scenario: faultinject.PermissionDenied, wantCode: codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx := &svc.ServiceContext{FaultInjection: faultinject.Config{
				Enabled: true,
				Rules:   map[string]string{id: string(tc.scenario)},
			}}
			resp, err := NewGetMessageRecordLogic(context.Background(), svcCtx).GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: id})
			if code := codeOf(err); code != tc.wantCode {
				t.Fatalf("code=%v want=%v", code, tc.wantCode)
			}
			if tc.wantNote != "" && (resp == nil || resp.Note != tc.wantNote) {
				actualNote := "<nil>"
				if resp != nil {
					actualNote = resp.Note
				}
				t.Fatalf("note=%q want=%q", actualNote, tc.wantNote)
			}
		})
	}
}

func TestFaultInjectionDeliveryAndConnectionScenarios(t *testing.T) {
	messageID := "665f1c0000000000000000ab"
	deliverySvc := &svc.ServiceContext{FaultInjection: faultinject.Config{
		Enabled: true,
		Rules:   map[string]string{messageID: string(faultinject.AckTimeout)},
	}}
	delivery, err := NewGetDeliveryTimelineLogic(context.Background(), deliverySvc).GetDeliveryTimeline(&operations.GetDeliveryTimelineRequest{MessageId: messageID})
	if err != nil || len(delivery.Events) != 1 || delivery.Events[0].EventType != "ack_timeout" {
		t.Fatalf("unexpected injected delivery response: %+v %v", delivery, err)
	}

	userID := "fault-user-offline"
	connectionSvc := &svc.ServiceContext{FaultInjection: faultinject.Config{
		Enabled: true,
		Rules:   map[string]string{userID: string(faultinject.ReceiverOffline)},
	}}
	connection, err := NewGetConnectionObservationsLogic(context.Background(), connectionSvc).GetConnectionObservations(&operations.GetConnectionObservationsRequest{UserId: userID})
	if err != nil || len(connection.Observations) != 1 || connection.Observations[0].State != "offline" {
		t.Fatalf("unexpected injected connection response: %+v %v", connection, err)
	}
}

func TestGetMessageRecord_TimeoutIsErrorNotNotFound(t *testing.T) {
	svcCtx := &svc.ServiceContext{
		ChatLogModel: &fakeChatLogModel{err: context.DeadlineExceeded},
		EventModel:   &fakeEventModel{},
	}
	l := NewGetMessageRecordLogic(context.Background(), svcCtx)

	resp, err := l.GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: "665f1c0000000000000000aa"})
	if err == nil {
		t.Fatalf("timeout must be error, not found=%v", resp)
	}
	if codeOf(err) != codes.DeadlineExceeded {
		t.Fatalf("timeout must be DEADLINE_EXCEEDED, got %v", err)
	}
	if resp != nil && resp.Found {
		t.Fatal("timeout must not claim found=false")
	}
}

func TestGetMessageRecord_Found(t *testing.T) {
	oid := "665f1c0000000000000000aa"
	svcCtx := &svc.ServiceContext{
		ChatLogModel: &fakeChatLogModel{log: &immodels.ChatLog{
			ConversationId: "c1",
			SendId:         "u1",
			RecvId:         "u2",
			ChatType:       constants.SingleChatType,
			SendTime:       42,
			ReadRecords:    []byte{1},
		}},
		EventModel: &fakeEventModel{events: []*observation.MessageEvent{{EventID: "e1"}}},
	}
	// ChatLog.ID 需要是 ObjectID
	svcCtx.ChatLogModel.(*fakeChatLogModel).log.ID = mustOID(oid)

	l := NewGetMessageRecordLogic(context.Background(), svcCtx)
	resp, err := l.GetMessageRecord(&operations.GetMessageRecordRequest{MessageId: oid})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Found || resp.SenderId != "u1" {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.ReadState != "known" {
		t.Fatalf("readState=%s", resp.ReadState)
	}
	if !resp.EventsAvailable {
		t.Fatal("eventsAvailable must be true")
	}
	// 脱敏：响应中不得有正文字段（proto 本身无 msgContent）
}

func TestDeriveReadState(t *testing.T) {
	st, note := deriveReadState(&immodels.ChatLog{
		ChatType:    constants.SingleChatType,
		ReadRecords: []byte{1},
	})
	if st != "known" || note == "" {
		t.Fatalf("single read: %s %s", st, note)
	}

	st, _ = deriveReadState(&immodels.ChatLog{
		ChatType:    constants.GroupChatType,
		ReadRecords: []byte{1, 2},
	})
	if st != "approximate" {
		t.Fatalf("group bitmap must be approximate, got %s", st)
	}

	st, _ = deriveReadState(&immodels.ChatLog{ChatType: constants.SingleChatType})
	if st != "unknown" {
		t.Fatalf("empty records must be unknown, got %s", st)
	}
}

// ---- capabilities ----

func TestGetCapabilities_HonestDefaults(t *testing.T) {
	s := &svc.ServiceContext{}
	resp, err := NewGetCapabilitiesLogic(context.Background(), s).GetCapabilities(&operations.GetCapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MessageRecord != "supported" {
		t.Fatalf("message_record=%s", resp.MessageRecord)
	}
	if resp.MessageTimeline != "unsupported" || resp.DeliveryEvents != "unsupported" ||
		resp.AckHistory != "unsupported" || resp.HistoricalConnection != "unsupported" {
		t.Fatalf("disabled observation must be unsupported: %+v", resp)
	}
	if resp.ObservationEnabled != "false" {
		t.Fatalf("ObservationEnabled=%s", resp.ObservationEnabled)
	}
}

func TestGetCapabilities_EnabledPartial(t *testing.T) {
	s := &svc.ServiceContext{}
	s.Config.DeliveryObservation.Enabled = true
	s.Config.DeliveryObservation.AckMode = observation.AckModeNoAck

	resp, err := NewGetCapabilitiesLogic(context.Background(), s).GetCapabilities(&operations.GetCapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MessageTimeline != "partial" {
		t.Fatalf("enabled => partial, got %s", resp.MessageTimeline)
	}
	if resp.AckHistory != "unsupported" {
		t.Fatalf("NoAck must keep ack_history unsupported, got %s", resp.AckHistory)
	}

	s.Config.DeliveryObservation.AckMode = observation.AckModeRigorAck
	resp, _ = NewGetCapabilitiesLogic(context.Background(), s).GetCapabilities(&operations.GetCapabilitiesRequest{})
	if resp.AckHistory != "partial" {
		t.Fatalf("RigorAck may claim partial, got %s", resp.AckHistory)
	}
}

// ---- timeline / connection validation ----

func TestGetMessageTimeline_Validation(t *testing.T) {
	l := &GetMessageTimelineLogic{ctx: context.Background(), svcCtx: &svc.ServiceContext{}}

	_, err := l.GetMessageTimeline(&operations.GetMessageTimelineRequest{MessageId: "bad"})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("bad messageId: %v", err)
	}
	_, err = l.GetMessageTimeline(&operations.GetMessageTimelineRequest{
		MessageId: "665f1c0000000000000000aa",
		StartTime: 20,
		EndTime:   10,
	})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("start>end: %v", err)
	}
	_, err = l.GetMessageTimeline(&operations.GetMessageTimelineRequest{
		MessageId: "665f1c0000000000000000aa",
		Limit:     9999,
	})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("over limit: %v", err)
	}
}

func TestGetConnectionObservations_Validation(t *testing.T) {
	l := &GetConnectionObservationsLogic{ctx: context.Background(), svcCtx: &svc.ServiceContext{}}
	_, err := l.GetConnectionObservations(&operations.GetConnectionObservationsRequest{
		UserId:    "u1",
		At:        100,
		StartTime: 1,
	})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("at + range must INVALID_ARGUMENT, got %v", err)
	}
	_, err = l.GetConnectionObservations(&operations.GetConnectionObservationsRequest{})
	if codeOf(err) != codes.InvalidArgument {
		t.Fatalf("missing userId must INVALID_ARGUMENT, got %v", err)
	}
}

func TestEvidenceOf(t *testing.T) {
	if evidenceOf(observation.EventDeliverySucceeded) != "gateway-write" {
		t.Fatal("delivery_succeeded must be gateway-write")
	}
	if evidenceOf(observation.EventAckReceived) != "transport-ack" {
		t.Fatal("ack_received must be transport-ack")
	}
	if evidenceOf(observation.EventReceiverOffline) != "offline-marker" {
		t.Fatal("receiver_offline must be offline-marker")
	}
}

func mustOID(hex string) bson.ObjectID {
	oid, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		panic(err)
	}
	return oid
}
