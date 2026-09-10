package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"seyra/backend/internal/auth"
)

func TestCreateDirectAndMessages(t *testing.T) {
	ctx := context.Background()
	users := auth.NewMemoryStore()
	created := time.Now().UTC()
	ada := auth.User{ID: "usr_ada", Username: "ada", PasswordHash: "x", CreatedAt: created}
	lin := auth.User{ID: "usr_lin", Username: "lin", PasswordHash: "x", CreatedAt: created}
	if err := users.CreateUser(ctx, ada); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateUser(ctx, lin); err != nil {
		t.Fatal(err)
	}
	hub := NewHub()
	svc := NewService(NewMemoryStore(), NewAuthDirectory(users), hub)

	if _, err := svc.CreateDirect(ctx, ada.ID, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := svc.CreateDirect(ctx, ada.ID, "ada"); !errors.Is(err, ErrCannotMessageSelf) {
		t.Fatalf("expected self rejection, got %v", err)
	}

	first, err := svc.CreateDirect(ctx, ada.ID, "lin")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateDirect(ctx, lin.ID, "ada")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same conversation, got %s and %s", first.ID, second.ID)
	}
	if first.Kind != KindDirect {
		t.Fatalf("expected direct kind, got %s", first.Kind)
	}

	maya := auth.User{ID: "usr_maya", Username: "maya", PasswordHash: "x", CreatedAt: created}
	if err := users.CreateUser(ctx, maya); err != nil {
		t.Fatal(err)
	}
	group, err := svc.CreateRoom(ctx, ada.ID, KindGroup, "Design Team", []string{"lin", "maya"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if group.Kind != KindGroup || group.Title != "Design Team" || group.MemberCount != 3 {
		t.Fatalf("unexpected group %+v", group)
	}
	members, err := svc.ListRoomMembers(ctx, ada.ID, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{}
	for _, m := range members {
		roles[m.User.ID] = m.Role
	}
	if roles[ada.ID] != RoleOwner || roles[lin.ID] != RoleMember {
		t.Fatalf("expected ada owner, got %+v", roles)
	}
	if err := svc.SetMemberRole(ctx, ada.ID, group.ID, lin.ID, RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetMemberRole(ctx, lin.ID, group.ID, maya.ID, RoleAdmin); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin cannot promote, got %v", err)
	}
	if err := svc.RemoveMember(ctx, lin.ID, group.ID, ada.ID); !errors.Is(err, ErrOwnerProtected) {
		t.Fatalf("cannot remove owner, got %v", err)
	}
	if err := svc.LeaveRoom(ctx, ada.ID, group.ID); !errors.Is(err, ErrOwnerProtected) {
		t.Fatalf("owner cannot leave, got %v", err)
	}
	if _, err := svc.AddMembers(ctx, lin.ID, group.ID, []string{"maya"}); err != nil && !errors.Is(err, ErrAlreadyMember) {
		// maya already in group; adding duplicate should be skipped, not fail
		if err != nil {
			t.Fatal(err)
		}
	}
	stranger := auth.User{ID: "usr_stranger", Username: "stranger", PasswordHash: "x", CreatedAt: created}
	if err := users.CreateUser(ctx, stranger); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMembers(ctx, maya.ID, group.ID, []string{"stranger"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member cannot add, got %v", err)
	}
	if _, err := svc.AddMembers(ctx, ada.ID, group.ID, []string{"stranger"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RemoveMember(ctx, ada.ID, group.ID, stranger.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.LeaveRoom(ctx, maya.ID, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListMessages(ctx, maya.ID, group.ID, "", 20); !errors.Is(err, ErrForbidden) {
		t.Fatalf("left member cannot read, got %v", err)
	}
	channel, err := svc.CreateRoom(ctx, ada.ID, KindChannel, "Seyra News", []string{"lin"}, VisibilityPublic)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := svc.List(ctx, lin.ID)
	if err != nil {
		t.Fatal(err)
	}
	foundGroup, foundChannel := false, false
	for _, item := range listed {
		if item.ID == group.ID {
			foundGroup = true
		}
		if item.ID == channel.ID {
			foundChannel = true
		}
	}
	if !foundGroup || !foundChannel {
		t.Fatalf("lin should see group and channel, got %+v", listed)
	}

	if _, err := svc.ListMessages(ctx, "usr_stranger", first.ID, "", 20); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}

	events, cancel := hub.Subscribe(lin.ID)
	defer cancel()

	msg, err := svc.SendMessage(ctx, ada.ID, first.ID, "Hello from Seyra", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if msg.SenderID != ada.ID || msg.Body != "Hello from Seyra" {
		t.Fatalf("unexpected message %+v", msg)
	}

	select {
	case event := <-events:
		if event.Type != EventMessageCreated {
			t.Fatalf("event %s", event.Type)
		}
	default:
		t.Fatal("expected realtime event")
	}

	page, err := svc.ListMessages(ctx, lin.ID, first.ID, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].Body != "Hello from Seyra" {
		t.Fatalf("expected persisted message, got %+v", page)
	}

	if _, err := svc.SendMessage(ctx, lin.ID, first.ID, "n1", "", "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendMessage(ctx, lin.ID, first.ID, "n2", "", "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendMessage(ctx, lin.ID, first.ID, "n3", "", "", false); err != nil {
		t.Fatal(err)
	}
	firstPage, err := svc.ListMessages(ctx, ada.ID, first.ID, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("expected page size 2, got %d", len(firstPage))
	}
	older, err := svc.ListMessages(ctx, ada.ID, first.ID, firstPage[0].ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(older) == 0 {
		t.Fatal("expected older messages")
	}

	edited, err := svc.EditMessage(ctx, ada.ID, first.ID, msg.ID, "Hello edited")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Body != "Hello edited" {
		t.Fatalf("edit %q", edited.Body)
	}
	reply, err := svc.SendMessage(ctx, lin.ID, first.ID, "replying", msg.ID, "", false)
	if err != nil || reply.ReplyToID != msg.ID {
		t.Fatalf("reply %+v %v", reply, err)
	}
	hits, err := svc.Search(ctx, ada.ID, "Hello", "messages")
	if err != nil {
		t.Fatal(err)
	}
	found, _ := hits["messages"].([]Message)
	if len(found) == 0 {
		t.Fatalf("expected search hits, got %#v", hits)
	}
	pub, err := svc.DiscoverChannels(ctx, ada.ID, "News")
	if err != nil || len(pub) == 0 {
		t.Fatalf("discover %v %+v", err, pub)
	}
}

func TestBotGrantsForwardBundleAndE2EEdit(t *testing.T) {
	ctx := context.Background()
	users := auth.NewMemoryStore()
	created := time.Now().UTC()
	ada := auth.User{ID: "usr_ada", Username: "ada", PasswordHash: "x", CreatedAt: created}
	lin := auth.User{ID: "usr_lin", Username: "lin", PasswordHash: "x", CreatedAt: created}
	maya := auth.User{ID: "usr_maya", Username: "maya", PasswordHash: "x", CreatedAt: created}
	botUser := auth.User{ID: "usr_bot", Username: "echo_bot", PasswordHash: "x", CreatedAt: created}
	for _, u := range []auth.User{ada, lin, maya, botUser} {
		if err := users.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(NewMemoryStore(), NewAuthDirectory(users), NewHub())
	group, err := svc.CreateRoom(ctx, ada.ID, KindGroup, "Team", []string{"lin"}, "")
	if err != nil {
		t.Fatal(err)
	}
	bot, _, err := svc.CreateBot(ctx, ada.ID, botUser.ID, "echo_bot")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GrantBot(ctx, ada.ID, bot.ID, group.ID, BotGrant{
		CanRead: false, CanSend: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendMessage(ctx, botUser.ID, group.ID, "pong", "", "", false); err != nil {
		t.Fatalf("bot send: %v", err)
	}
	if _, err := svc.SendMessage(ctx, ada.ID, group.ID, "/help", "", "", false); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	foundHelp := false
	for time.Now().Before(deadline) {
		msgs, err := svc.ListMessages(ctx, ada.ID, group.ID, "", 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range msgs {
			if m.Body == "Commands: /ping, /help, /whoami" {
				foundHelp = true
			}
		}
		if foundHelp {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !foundHelp {
		t.Fatal("bot /help reply missing")
	}
	if _, err := svc.ListMessages(ctx, botUser.ID, group.ID, "", 20); !errors.Is(err, ErrForbidden) {
		t.Fatalf("can_read=false should forbid list, got %v", err)
	}
	if _, err := svc.AddMembers(ctx, lin.ID, group.ID, []string{"maya"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ordinary member must not add members, got %v", err)
	}

	direct, err := svc.CreateDirect(ctx, ada.ID, "lin")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.FetchBundle(ctx, ada.ID, maya.ID, "1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("bundle without 1:1 should be forbidden, got %v", err)
	}
	other, err := svc.CreateDirect(ctx, ada.ID, "maya")
	if err != nil {
		t.Fatal(err)
	}
	src, err := svc.SendMessage(ctx, ada.ID, direct.ID, "forward me", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := svc.ForwardMessage(ctx, ada.ID, direct.ID, src.ID, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copied.ForwardedFromID != src.ID {
		t.Fatalf("forwarded_from_id not persisted: %+v", copied)
	}
	got, err := svc.store.GetMessage(ctx, copied.ID)
	if err != nil || got.ForwardedFromID != src.ID {
		t.Fatalf("store forward %v %+v", err, got)
	}

	enc, err := svc.SendMessage(ctx, ada.ID, direct.ID, "cipher", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EditMessage(ctx, ada.ID, direct.ID, enc.ID, "plaintext leak"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("e2e edit should be forbidden, got %v", err)
	}
	if _, _, err := svc.FetchBundle(ctx, ada.ID, lin.ID, "1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bundle with 1:1 but no keys should be not found, got %v", err)
	}

	channel, err := svc.CreateRoom(ctx, ada.ID, KindChannel, "News", []string{"lin"}, VisibilityPublic)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendMessage(ctx, ada.ID, channel.ID, "cipher", "", "", true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("channel e2e should be forbidden, got %v", err)
	}

	hub := NewHub()
	typed := make(chan Event, 1)
	events, cancel := hub.Subscribe(lin.ID)
	defer cancel()
	go func() {
		typed <- <-events
	}()
	svc2 := NewService(svc.store, svc.directory, hub)
	if err := svc2.PublishTyping(ctx, ada.ID, direct.ID, true); err != nil {
		t.Fatal(err)
	}
	gotEvt := <-typed
	if gotEvt.Type != EventTyping {
		t.Fatalf("typing event %s", gotEvt.Type)
	}
	if err := svc.PutPrivacy(ctx, lin.ID, PrivacySettings{UserID: lin.ID, LastSeenVisible: false, ReadReceipts: true, TypingVisible: true, ProfileVisible: true, NotificationPreview: true, PhotoVisible: true}); err != nil {
		t.Fatal(err)
	}
	_ = svc.TouchPresence(ctx, lin.ID)
	seen, err := svc.PeerLastSeen(ctx, ada.ID, lin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if seen != nil {
		t.Fatal("hidden last seen leaked")
	}

	keysGroup, err := svc.CreateRoom(ctx, ada.ID, KindGroup, "Keys", []string{"lin"}, VisibilityPrivate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadAttachment(ctx, ada.ID, keysGroup.ID, "encrypted.bin", "application/octet-stream", strings.NewReader("cipher"), 6, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("group e2e upload should be forbidden, got %v", err)
	}
	plainAtt, err := svc.UploadAttachment(ctx, ada.ID, direct.ID, "note.txt", "text/plain", strings.NewReader("hello"), 5, false)
	if err != nil {
		t.Fatal(err)
	}
	withFile, err := svc.SendMessage(ctx, ada.ID, direct.ID, "note.txt", "", plainAtt.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ForwardMessage(ctx, ada.ID, direct.ID, withFile.ID, other.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("forwarding attachment messages should be forbidden, got %v", err)
	}
}
