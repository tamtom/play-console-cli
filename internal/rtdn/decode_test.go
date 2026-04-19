package rtdn

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func wrapEnvelope(t *testing.T, inner map[string]any) []byte {
	t.Helper()
	innerBytes, err := json.Marshal(inner)
	if err != nil {
		t.Fatal(err)
	}
	env := PubsubEnvelope{}
	env.Message.Data = base64.StdEncoding.EncodeToString(innerBytes)
	env.Message.MessageID = "123"
	env.Message.PublishTime = "2026-04-19T10:00:00Z"
	env.Subscription = "projects/p/subscriptions/s"
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDecodeSubscriptionRenewed(t *testing.T) {
	data := wrapEnvelope(t, map[string]any{
		"version":         "1.0",
		"packageName":     "com.example.app",
		"eventTimeMillis": "1700000000000",
		"subscriptionNotification": map[string]any{
			"version":          "1.0",
			"notificationType": float64(2),
			"purchaseToken":    "token-abc",
			"subscriptionId":   "premium",
		},
	})
	d, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindSubscription {
		t.Errorf("kind=%s want subscription", d.Kind)
	}
	if d.Subscription == nil || d.Subscription.NotificationType != 2 {
		t.Fatalf("unexpected subscription payload: %+v", d.Subscription)
	}
	if d.Subscription.NotificationName != "SUBSCRIPTION_RENEWED" {
		t.Errorf("name=%s want SUBSCRIPTION_RENEWED", d.Subscription.NotificationName)
	}
}

func TestDecodeOneTime(t *testing.T) {
	data := wrapEnvelope(t, map[string]any{
		"packageName": "com.example.app",
		"oneTimeProductNotification": map[string]any{
			"notificationType": float64(1),
			"purchaseToken":    "t",
			"sku":              "coins_100",
		},
	})
	d, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindOneTime {
		t.Errorf("kind=%s want one_time_product", d.Kind)
	}
	if d.OneTimeProduct.NotificationName != "ONE_TIME_PRODUCT_PURCHASED" {
		t.Errorf("name=%s", d.OneTimeProduct.NotificationName)
	}
}

func TestDecodeVoided(t *testing.T) {
	data := wrapEnvelope(t, map[string]any{
		"voidedPurchaseNotification": map[string]any{
			"purchaseToken": "t",
			"orderId":       "order-1",
			"productType":   float64(1),
			"refundType":    float64(1),
		},
	})
	d, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindVoided || d.Voided.OrderID != "order-1" {
		t.Fatalf("unexpected %+v", d)
	}
}

func TestDecodeTestNotification(t *testing.T) {
	data := wrapEnvelope(t, map[string]any{
		"testNotification": map[string]any{"version": "1.0"},
	})
	d, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindTest {
		t.Errorf("kind=%s", d.Kind)
	}
}

func TestDecodeRawInner(t *testing.T) {
	// Pass raw inner JSON (no envelope).
	inner := map[string]any{
		"packageName": "com.ex",
		"subscriptionNotification": map[string]any{
			"notificationType": float64(4),
		},
	}
	b, _ := json.Marshal(inner)
	d, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindSubscription {
		t.Errorf("kind=%s", d.Kind)
	}
	if d.Subscription.NotificationName != "SUBSCRIPTION_PURCHASED" {
		t.Errorf("name=%s", d.Subscription.NotificationName)
	}
}

func TestDecodeRejectsEmpty(t *testing.T) {
	if _, err := Decode(nil); err == nil {
		t.Error("expected error")
	}
	if _, err := Decode([]byte("   ")); err == nil {
		t.Error("expected error for whitespace only")
	}
}

func TestDecodeRejectsBadJSON(t *testing.T) {
	if _, err := Decode([]byte("not-json")); err == nil {
		t.Error("expected error")
	}
}

func TestDecodeRejectsBadBase64(t *testing.T) {
	env := PubsubEnvelope{}
	env.Message.Data = "!!!not-base64!!!"
	b, _ := json.Marshal(env)
	if _, err := Decode(b); err == nil {
		t.Error("expected error for bad base64")
	}
}

func TestUnknownKind(t *testing.T) {
	data := wrapEnvelope(t, map[string]any{
		"packageName": "com.ex",
	})
	d, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindUnknown {
		t.Errorf("kind=%s", d.Kind)
	}
}
