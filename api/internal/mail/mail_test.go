package mail

import (
	"context"
	"errors"
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/testdb"
)

func TestValidateTemplateHTML(t *testing.T) {
	tpl := Template{RequiredTags: []string{"site_name", "unsubscribe_url"}}

	if err := ValidateTemplateHTML(tpl, "{{site_name}}", `<a href="{{unsubscribe_url}}">sair</a>`); err != nil {
		t.Fatalf("valid template rejected: %v", err)
	}
	err := ValidateTemplateHTML(tpl, "{{site_name}}", "<p>sem link</p>")
	if !errors.Is(err, ErrTemplate) {
		t.Fatalf("err = %v, want ErrTemplate", err)
	}
}

func TestRender(t *testing.T) {
	got := Render("{{a}} e {{b}} e {{a}} e {{c}}", map[string]string{"a": "1", "b": "2"})
	if want := "1 e 2 e 1 e {{c}}"; got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Ana@Example.ORG "); got != "ana@example.org" {
		t.Fatalf("normalizeEmail = %q", got)
	}
}

func TestIsInvalidRecipient(t *testing.T) {
	for _, msg := range []string{"550 5.1.1 rejected", "Invalid recipient", "User unknown", "Mailbox unavailable"} {
		if !isInvalidRecipient(errors.New(msg)) {
			t.Fatalf("%q should be an invalid recipient", msg)
		}
	}
	if isInvalidRecipient(errors.New("421 try again later")) {
		t.Fatal("temporary failure treated as invalid recipient")
	}
}

func TestPqArray(t *testing.T) {
	if got := pqArray([]string{"a", `b"c`}); got != `{"a","b\"c"}` {
		t.Fatalf("pqArray = %v", got)
	}
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	for _, bad := range []string{"sem-arroba", "a b@gmail.com", "@gmail.com", "ana@localhost"} {
		if _, err := r.Subscribe(ctx, bad); !errors.Is(err, ErrInvalidEmail) {
			t.Fatalf("Subscribe(%q) err = %v, want ErrInvalidEmail", bad, err)
		}
	}

	// gmail.com is a trusted domain seeded by migration, so no DNS lookup happens.
	for i := 0; i < 2; i++ {
		res, err := r.Subscribe(ctx, " Ana@Gmail.com ")
		if err != nil || !res.OK {
			t.Fatalf("subscribe #%d = %+v, %v", i+1, res, err)
		}
	}
	n, err := r.ActiveSubscriberCount(ctx)
	if err != nil || n != 1 {
		t.Fatalf("active subscribers = %d, %v; want 1", n, err)
	}

	var token string
	if err := r.DB.QueryRow(`SELECT unsubscribe_token FROM email_subscribers WHERE email_normalized = 'ana@gmail.com'`).Scan(&token); err != nil {
		t.Fatalf("read token: %v", err)
	}
	if err := r.Unsubscribe(ctx, token); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if err := r.Unsubscribe(ctx, "token-inexistente"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown token err = %v, want ErrNotFound", err)
	}
	if n, _ := r.ActiveSubscriberCount(ctx); n != 0 {
		t.Fatalf("active subscribers after unsubscribe = %d, want 0", n)
	}
}

func TestFindShortagesUsesPerFoodThreshold(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	rice := testdb.Food(t, r.DB, "arroz", "Arroz", "kg")
	beans := testdb.Food(t, r.DB, "feijao", "Feijão", "kg")
	for _, id := range []int64{rice, beans} {
		if _, err := r.DB.Exec(`INSERT INTO stock_movements(food_id, movement_type, quantity) VALUES ($1, 'IN', 5)`, id); err != nil {
			t.Fatalf("stock in: %v", err)
		}
	}
	if err := r.SetFoodThreshold(ctx, rice, quantity.Must("10")); err != nil {
		t.Fatalf("set threshold: %v", err)
	}

	shortages, err := r.FindShortages(ctx, quantity.Must("2"))
	if err != nil {
		t.Fatalf("find shortages: %v", err)
	}
	if len(shortages) != 1 || shortages[0].Name != "Arroz" {
		t.Fatalf("shortages = %+v, want only Arroz", shortages)
	}
	if shortages[0].Missing.Cmp(quantity.Must("5")) != 0 {
		t.Fatalf("missing = %s, want 5", shortages[0].Missing)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	cfg := SMTPConfig{Host: "smtp.example.org", Port: "587", User: "u", Password: "p", From: "no-reply@example.org"}
	if err := r.SaveSMTP(ctx, cfg); err != nil {
		t.Fatalf("save smtp: %v", err)
	}
	got, err := r.LoadSMTP(ctx)
	if err != nil || got != cfg {
		t.Fatalf("load smtp = %+v, %v; want %+v", got, err, cfg)
	}

	defaults := r.LoadAlertSettings(ctx)
	if defaults.AutoEnabled || defaults.AutoFrequency != "daily" {
		t.Fatalf("default alert settings = %+v", defaults)
	}
	want := AlertSettings{AutoEnabled: true, AutoFrequency: "weekly", GlobalThreshold: quantity.Must("3")}
	if err := r.SaveAlertSettings(ctx, want); err != nil {
		t.Fatalf("save alert settings: %v", err)
	}
	got2 := r.LoadAlertSettings(ctx)
	if !got2.AutoEnabled || got2.AutoFrequency != "weekly" || got2.GlobalThreshold.Cmp(want.GlobalThreshold) != 0 {
		t.Fatalf("alert settings = %+v, want %+v", got2, want)
	}
}

func TestUpdateTemplateRequiresTags(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	if _, err := r.UpdateTemplate(ctx, "shortage_alert", "Faltas", "<p>sem tags</p>"); !errors.Is(err, ErrTemplate) {
		t.Fatalf("err = %v, want ErrTemplate", err)
	}
	if _, err := r.UpdateTemplate(ctx, "nao_existe", "x", "y"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown template err = %v, want ErrNotFound", err)
	}
	body := `<p>{{site_name}}</p>{{shortage_list}}<a href="{{unsubscribe_url}}">sair</a>`
	tpl, err := r.UpdateTemplate(ctx, "shortage_alert", "{{site_name}}: faltas", body)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if tpl.BodyHTML != body {
		t.Fatalf("body not saved: %q", tpl.BodyHTML)
	}
}

func TestSendShortageAlertWithoutShortagesOrSMTP(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	res, err := r.SendShortageAlert(ctx, "Megumi Kura", "http://localhost", "manual", nil)
	if err != nil || res.SkippedReason != "no_shortages" {
		t.Fatalf("no shortages = %+v, %v", res, err)
	}

	items := []ShortageItem{{Name: "Arroz", Unit: "kg", Stock: quantity.Zero()}}
	if _, err := r.SendShortageAlert(ctx, "Megumi Kura", "http://localhost", "manual", items); !errors.Is(err, ErrSMTP) {
		t.Fatalf("without smtp err = %v, want ErrSMTP", err)
	}
}
