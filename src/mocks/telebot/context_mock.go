// gen hack 
package mock_telebot

import (
	reflect "reflect"
	time "time"

	src "gopkg.in/telebot.v4"
	gomock "go.uber.org/mock/gomock"
)

// MockContext is a mock of Context interface.
type MockContext struct {
	ctrl     *gomock.Controller
	recorder *MockContextMockRecorder
	isgomock struct{}
}

// MockContextMockRecorder is the mock recorder for MockContext.
type MockContextMockRecorder struct {
	mock *MockContext
}

// NewMockContext creates a new mock instance.
func NewMockContext(ctrl *gomock.Controller) *MockContext {
	mock := &MockContext{ctrl: ctrl}
	mock.recorder = &MockContextMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockContext) EXPECT() *MockContextMockRecorder {
	return m.recorder
}

// Accept mocks base method.
func (m *MockContext) Accept(errorMessage ...string) error {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range errorMessage {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Accept", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Accept indicates an expected call of Accept.
func (mr *MockContextMockRecorder) Accept(errorMessage ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Accept", reflect.TypeOf((*MockContext)(nil).Accept), errorMessage...)
}

// Answer mocks base method.
func (m *MockContext) Answer(resp *src.QueryResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Answer", resp)
	ret0, _ := ret[0].(error)
	return ret0
}

// Answer indicates an expected call of Answer.
func (mr *MockContextMockRecorder) Answer(resp any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Answer", reflect.TypeOf((*MockContext)(nil).Answer), resp)
}

// Args mocks base method.
func (m *MockContext) Args() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Args")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Args indicates an expected call of Args.
func (mr *MockContextMockRecorder) Args() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Args", reflect.TypeOf((*MockContext)(nil).Args))
}

// Boost mocks base method.
func (m *MockContext) Boost() *src.BoostUpdated {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Boost")
	ret0, _ := ret[0].(*src.BoostUpdated)
	return ret0
}

// Boost indicates an expected call of Boost.
func (mr *MockContextMockRecorder) Boost() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Boost", reflect.TypeOf((*MockContext)(nil).Boost))
}

// BoostRemoved mocks base method.
func (m *MockContext) BoostRemoved() *src.BoostRemoved {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BoostRemoved")
	ret0, _ := ret[0].(*src.BoostRemoved)
	return ret0
}

// BoostRemoved indicates an expected call of BoostRemoved.
func (mr *MockContextMockRecorder) BoostRemoved() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BoostRemoved", reflect.TypeOf((*MockContext)(nil).BoostRemoved))
}

// Bot mocks base method.
func (m *MockContext) Bot() src.API {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Bot")
	ret0, _ := ret[0].(src.API)
	return ret0
}

// Bot indicates an expected call of Bot.
func (mr *MockContextMockRecorder) Bot() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Bot", reflect.TypeOf((*MockContext)(nil).Bot))
}

// Callback mocks base method.
func (m *MockContext) Callback() *src.Callback {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Callback")
	ret0, _ := ret[0].(*src.Callback)
	return ret0
}

// Callback indicates an expected call of Callback.
func (mr *MockContextMockRecorder) Callback() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Callback", reflect.TypeOf((*MockContext)(nil).Callback))
}

// Chat mocks base method.
func (m *MockContext) Chat() *src.Chat {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chat")
	ret0, _ := ret[0].(*src.Chat)
	return ret0
}

// Chat indicates an expected call of Chat.
func (mr *MockContextMockRecorder) Chat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chat", reflect.TypeOf((*MockContext)(nil).Chat))
}

// ChatJoinRequest mocks base method.
func (m *MockContext) ChatJoinRequest() *src.ChatJoinRequest {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChatJoinRequest")
	ret0, _ := ret[0].(*src.ChatJoinRequest)
	return ret0
}

// ChatJoinRequest indicates an expected call of ChatJoinRequest.
func (mr *MockContextMockRecorder) ChatJoinRequest() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChatJoinRequest", reflect.TypeOf((*MockContext)(nil).ChatJoinRequest))
}

// ChatMember mocks base method.
func (m *MockContext) ChatMember() *src.ChatMemberUpdate {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChatMember")
	ret0, _ := ret[0].(*src.ChatMemberUpdate)
	return ret0
}

// ChatMember indicates an expected call of ChatMember.
func (mr *MockContextMockRecorder) ChatMember() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChatMember", reflect.TypeOf((*MockContext)(nil).ChatMember))
}

// Data mocks base method.
func (m *MockContext) Data() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Data")
	ret0, _ := ret[0].(string)
	return ret0
}

// Data indicates an expected call of Data.
func (mr *MockContextMockRecorder) Data() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Data", reflect.TypeOf((*MockContext)(nil).Data))
}

// Delete mocks base method.
func (m *MockContext) Delete() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete")
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockContextMockRecorder) Delete() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockContext)(nil).Delete))
}

// DeleteAfter mocks base method.
func (m *MockContext) DeleteAfter(d time.Duration) *time.Timer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAfter", d)
	ret0, _ := ret[0].(*time.Timer)
	return ret0
}

// DeleteAfter indicates an expected call of DeleteAfter.
func (mr *MockContextMockRecorder) DeleteAfter(d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAfter", reflect.TypeOf((*MockContext)(nil).DeleteAfter), d)
}

// Edit mocks base method.
func (m *MockContext) Edit(what any, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{what}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Edit", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Edit indicates an expected call of Edit.
func (mr *MockContextMockRecorder) Edit(what any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{what}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Edit", reflect.TypeOf((*MockContext)(nil).Edit), varargs...)
}

// EditCaption mocks base method.
func (m *MockContext) EditCaption(caption string, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{caption}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "EditCaption", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// EditCaption indicates an expected call of EditCaption.
func (mr *MockContextMockRecorder) EditCaption(caption any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{caption}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EditCaption", reflect.TypeOf((*MockContext)(nil).EditCaption), varargs...)
}

// EditOrReply mocks base method.
func (m *MockContext) EditOrReply(what any, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{what}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "EditOrReply", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// EditOrReply indicates an expected call of EditOrReply.
func (mr *MockContextMockRecorder) EditOrReply(what any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{what}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EditOrReply", reflect.TypeOf((*MockContext)(nil).EditOrReply), varargs...)
}

// EditOrSend mocks base method.
func (m *MockContext) EditOrSend(what any, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{what}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "EditOrSend", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// EditOrSend indicates an expected call of EditOrSend.
func (mr *MockContextMockRecorder) EditOrSend(what any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{what}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EditOrSend", reflect.TypeOf((*MockContext)(nil).EditOrSend), varargs...)
}

// Entities mocks base method.
func (m *MockContext) Entities() src.Entities {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Entities")
	ret0, _ := ret[0].(src.Entities)
	return ret0
}

// Entities indicates an expected call of Entities.
func (mr *MockContextMockRecorder) Entities() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Entities", reflect.TypeOf((*MockContext)(nil).Entities))
}

// Forward mocks base method.
func (m *MockContext) Forward(msg src.Editable, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{msg}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Forward", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Forward indicates an expected call of Forward.
func (mr *MockContextMockRecorder) Forward(msg any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{msg}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Forward", reflect.TypeOf((*MockContext)(nil).Forward), varargs...)
}

// ForwardTo mocks base method.
func (m *MockContext) ForwardTo(to src.Recipient, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{to}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ForwardTo", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// ForwardTo indicates an expected call of ForwardTo.
func (mr *MockContextMockRecorder) ForwardTo(to any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{to}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForwardTo", reflect.TypeOf((*MockContext)(nil).ForwardTo), varargs...)
}

// Get mocks base method.
func (m *MockContext) Get(key string) any {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", key)
	ret0, _ := ret[0].(any)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockContextMockRecorder) Get(key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockContext)(nil).Get), key)
}

// InlineResult mocks base method.
func (m *MockContext) InlineResult() *src.InlineResult {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InlineResult")
	ret0, _ := ret[0].(*src.InlineResult)
	return ret0
}

// InlineResult indicates an expected call of InlineResult.
func (mr *MockContextMockRecorder) InlineResult() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InlineResult", reflect.TypeOf((*MockContext)(nil).InlineResult))
}

// Message mocks base method.
func (m *MockContext) Message() *src.Message {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Message")
	ret0, _ := ret[0].(*src.Message)
	return ret0
}

// Message indicates an expected call of Message.
func (mr *MockContextMockRecorder) Message() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Message", reflect.TypeOf((*MockContext)(nil).Message))
}

// Migration mocks base method.
func (m *MockContext) Migration() (int64, int64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Migration")
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(int64)
	return ret0, ret1
}

// Migration indicates an expected call of Migration.
func (mr *MockContextMockRecorder) Migration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Migration", reflect.TypeOf((*MockContext)(nil).Migration))
}

// Notify mocks base method.
func (m *MockContext) Notify(action src.ChatAction) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Notify", action)
	ret0, _ := ret[0].(error)
	return ret0
}

// Notify indicates an expected call of Notify.
func (mr *MockContextMockRecorder) Notify(action any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Notify", reflect.TypeOf((*MockContext)(nil).Notify), action)
}

// Payment mocks base method.
func (m *MockContext) Payment() *src.Payment {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Payment")
	ret0, _ := ret[0].(*src.Payment)
	return ret0
}

// Payment indicates an expected call of Payment.
func (mr *MockContextMockRecorder) Payment() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Payment", reflect.TypeOf((*MockContext)(nil).Payment))
}

// Poll mocks base method.
func (m *MockContext) Poll() *src.Poll {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Poll")
	ret0, _ := ret[0].(*src.Poll)
	return ret0
}

// Poll indicates an expected call of Poll.
func (mr *MockContextMockRecorder) Poll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Poll", reflect.TypeOf((*MockContext)(nil).Poll))
}

// PollAnswer mocks base method.
func (m *MockContext) PollAnswer() *src.PollAnswer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PollAnswer")
	ret0, _ := ret[0].(*src.PollAnswer)
	return ret0
}

// PollAnswer indicates an expected call of PollAnswer.
func (mr *MockContextMockRecorder) PollAnswer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PollAnswer", reflect.TypeOf((*MockContext)(nil).PollAnswer))
}

// PreCheckoutQuery mocks base method.
func (m *MockContext) PreCheckoutQuery() *src.PreCheckoutQuery {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreCheckoutQuery")
	ret0, _ := ret[0].(*src.PreCheckoutQuery)
	return ret0
}

// PreCheckoutQuery indicates an expected call of PreCheckoutQuery.
func (mr *MockContextMockRecorder) PreCheckoutQuery() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreCheckoutQuery", reflect.TypeOf((*MockContext)(nil).PreCheckoutQuery))
}

// Query mocks base method.
func (m *MockContext) Query() *src.Query {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query")
	ret0, _ := ret[0].(*src.Query)
	return ret0
}

// Query indicates an expected call of Query.
func (mr *MockContextMockRecorder) Query() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockContext)(nil).Query))
}

// Recipient mocks base method.
func (m *MockContext) Recipient() src.Recipient {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recipient")
	ret0, _ := ret[0].(src.Recipient)
	return ret0
}

// Recipient indicates an expected call of Recipient.
func (mr *MockContextMockRecorder) Recipient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recipient", reflect.TypeOf((*MockContext)(nil).Recipient))
}

// Reply mocks base method.
func (m *MockContext) Reply(what any, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{what}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Reply", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Reply indicates an expected call of Reply.
func (mr *MockContextMockRecorder) Reply(what any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{what}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reply", reflect.TypeOf((*MockContext)(nil).Reply), varargs...)
}

// Respond mocks base method.
func (m *MockContext) Respond(resp ...*src.CallbackResponse) error {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range resp {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Respond", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Respond indicates an expected call of Respond.
func (mr *MockContextMockRecorder) Respond(resp ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Respond", reflect.TypeOf((*MockContext)(nil).Respond), resp...)
}

// RespondAlert mocks base method.
func (m *MockContext) RespondAlert(text string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RespondAlert", text)
	ret0, _ := ret[0].(error)
	return ret0
}

// RespondAlert indicates an expected call of RespondAlert.
func (mr *MockContextMockRecorder) RespondAlert(text any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RespondAlert", reflect.TypeOf((*MockContext)(nil).RespondAlert), text)
}

// RespondText mocks base method.
func (m *MockContext) RespondText(text string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RespondText", text)
	ret0, _ := ret[0].(error)
	return ret0
}

// RespondText indicates an expected call of RespondText.
func (mr *MockContextMockRecorder) RespondText(text any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RespondText", reflect.TypeOf((*MockContext)(nil).RespondText), text)
}

// Send mocks base method.
func (m *MockContext) Send(what any, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{what}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Send", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockContextMockRecorder) Send(what any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{what}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockContext)(nil).Send), varargs...)
}

// SendAlbum mocks base method.
func (m *MockContext) SendAlbum(a src.Album, opts ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{a}
	for _, a_2 := range opts {
		varargs = append(varargs, a_2)
	}
	ret := m.ctrl.Call(m, "SendAlbum", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendAlbum indicates an expected call of SendAlbum.
func (mr *MockContextMockRecorder) SendAlbum(a any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{a}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendAlbum", reflect.TypeOf((*MockContext)(nil).SendAlbum), varargs...)
}

// Sender mocks base method.
func (m *MockContext) Sender() *src.User {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sender")
	ret0, _ := ret[0].(*src.User)
	return ret0
}

// Sender indicates an expected call of Sender.
func (mr *MockContextMockRecorder) Sender() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sender", reflect.TypeOf((*MockContext)(nil).Sender))
}

// Set mocks base method.
func (m *MockContext) Set(key string, val any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Set", key, val)
}

// Set indicates an expected call of Set.
func (mr *MockContextMockRecorder) Set(key, val any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockContext)(nil).Set), key, val)
}

// Ship mocks base method.
func (m *MockContext) Ship(what ...any) error {
	m.ctrl.T.Helper()
	varargs := []any{}
	for _, a := range what {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Ship", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Ship indicates an expected call of Ship.
func (mr *MockContextMockRecorder) Ship(what ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ship", reflect.TypeOf((*MockContext)(nil).Ship), what...)
}

// ShippingQuery mocks base method.
func (m *MockContext) ShippingQuery() *src.ShippingQuery {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShippingQuery")
	ret0, _ := ret[0].(*src.ShippingQuery)
	return ret0
}

// ShippingQuery indicates an expected call of ShippingQuery.
func (mr *MockContextMockRecorder) ShippingQuery() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShippingQuery", reflect.TypeOf((*MockContext)(nil).ShippingQuery))
}

// Text mocks base method.
func (m *MockContext) Text() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Text")
	ret0, _ := ret[0].(string)
	return ret0
}

// Text indicates an expected call of Text.
func (mr *MockContextMockRecorder) Text() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Text", reflect.TypeOf((*MockContext)(nil).Text))
}

// Topic mocks base method.
func (m *MockContext) Topic() *src.Topic {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Topic")
	ret0, _ := ret[0].(*src.Topic)
	return ret0
}

// Topic indicates an expected call of Topic.
func (mr *MockContextMockRecorder) Topic() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Topic", reflect.TypeOf((*MockContext)(nil).Topic))
}

// Update mocks base method.
func (m *MockContext) Update() src.Update {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update")
	ret0, _ := ret[0].(src.Update)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockContextMockRecorder) Update() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockContext)(nil).Update))
}
