// Package rtdn implements Real-Time Developer Notification payload parsing
// and helpers for gplay's `rtdn` command family. Decoding is offline and
// deterministic; setup/status use gcloud subprocesses when wired from the CLI.
package rtdn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// PubsubEnvelope mirrors the JSON payload Google Pub/Sub POSTs to a push
// endpoint. For a rich reference see:
// https://developer.android.com/google/play/billing/rtdn-reference
type PubsubEnvelope struct {
	Message struct {
		Attributes  map[string]string `json:"attributes,omitempty"`
		Data        string            `json:"data"` // base64-encoded JSON
		MessageID   string            `json:"messageId,omitempty"`
		PublishTime string            `json:"publishTime,omitempty"`
	} `json:"message"`
	Subscription string `json:"subscription,omitempty"`
}

// NotificationKind enumerates the top-level shape of the decoded payload.
type NotificationKind string

const (
	KindSubscription NotificationKind = "subscription"
	KindOneTime      NotificationKind = "one_time_product"
	KindVoided       NotificationKind = "voided_purchase"
	KindTest         NotificationKind = "test"
	KindUnknown      NotificationKind = "unknown"
)

// DecodedNotification is the pretty, typed form of an RTDN payload.
type DecodedNotification struct {
	Kind             NotificationKind       `json:"kind"`
	Version          string                 `json:"version,omitempty"`
	PackageName      string                 `json:"package_name,omitempty"`
	EventTimeMillis  string                 `json:"event_time_millis,omitempty"`
	Subscription     *SubscriptionPayload   `json:"subscription,omitempty"`
	OneTimeProduct   *OneTimeProductPayload `json:"one_time_product,omitempty"`
	Voided           *VoidedPurchasePayload `json:"voided_purchase,omitempty"`
	TestNotification map[string]any         `json:"test_notification,omitempty"`
	Raw              map[string]any         `json:"raw,omitempty"`
}

// SubscriptionPayload is a SubscriptionNotification field.
type SubscriptionPayload struct {
	Version          string `json:"version,omitempty"`
	NotificationType int    `json:"notification_type"`
	PurchaseToken    string `json:"purchase_token,omitempty"`
	SubscriptionID   string `json:"subscription_id,omitempty"`
	NotificationName string `json:"notification_name,omitempty"` // human label
}

// OneTimeProductPayload is a OneTimeProductNotification field.
type OneTimeProductPayload struct {
	Version          string `json:"version,omitempty"`
	NotificationType int    `json:"notification_type"`
	PurchaseToken    string `json:"purchase_token,omitempty"`
	SKU              string `json:"sku,omitempty"`
	NotificationName string `json:"notification_name,omitempty"`
}

// VoidedPurchasePayload is a VoidedPurchaseNotification field.
type VoidedPurchasePayload struct {
	PurchaseToken string `json:"purchase_token,omitempty"`
	OrderID       string `json:"order_id,omitempty"`
	ProductType   int    `json:"product_type,omitempty"`
	RefundType    int    `json:"refund_type,omitempty"`
}

// Subscription notification type labels, per Play Billing docs.
var subscriptionNames = map[int]string{
	1:  "SUBSCRIPTION_RECOVERED",
	2:  "SUBSCRIPTION_RENEWED",
	3:  "SUBSCRIPTION_CANCELED",
	4:  "SUBSCRIPTION_PURCHASED",
	5:  "SUBSCRIPTION_ON_HOLD",
	6:  "SUBSCRIPTION_IN_GRACE_PERIOD",
	7:  "SUBSCRIPTION_RESTARTED",
	8:  "SUBSCRIPTION_PRICE_CHANGE_CONFIRMED",
	9:  "SUBSCRIPTION_DEFERRED",
	10: "SUBSCRIPTION_PAUSED",
	11: "SUBSCRIPTION_PAUSE_SCHEDULE_CHANGED",
	12: "SUBSCRIPTION_REVOKED",
	13: "SUBSCRIPTION_EXPIRED",
	20: "SUBSCRIPTION_PENDING_PURCHASE_CANCELED",
}

var oneTimeNames = map[int]string{
	1: "ONE_TIME_PRODUCT_PURCHASED",
	2: "ONE_TIME_PRODUCT_CANCELED",
}

// Decode takes a Pub/Sub envelope (JSON string or bytes) and returns a typed
// notification plus the decoded inner payload.
func Decode(data []byte) (*DecodedNotification, error) {
	data = bytes(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, errors.New("empty payload")
	}

	var env PubsubEnvelope
	// Accept either a full Pub/Sub envelope, or the inner data already decoded.
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}
	if env.Message.Data == "" {
		// Might be raw inner payload; try that.
		return decodeInner(data)
	}
	inner, err := base64.StdEncoding.DecodeString(env.Message.Data)
	if err != nil {
		return nil, fmt.Errorf("base64 decode message.data: %w", err)
	}
	dec, err := decodeInner(inner)
	if err != nil {
		return nil, err
	}
	return dec, nil
}

// decodeInner parses the base64-decoded JSON payload into a typed struct.
func decodeInner(data []byte) (*DecodedNotification, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse inner JSON: %w", err)
	}

	d := &DecodedNotification{
		Kind: KindUnknown,
		Raw:  raw,
	}
	if v, ok := raw["version"].(string); ok {
		d.Version = v
	}
	if v, ok := raw["packageName"].(string); ok {
		d.PackageName = v
	}
	if v, ok := raw["eventTimeMillis"].(string); ok {
		d.EventTimeMillis = v
	}

	// Pick the first populated typed payload.
	if sub, ok := raw["subscriptionNotification"].(map[string]any); ok {
		d.Kind = KindSubscription
		d.Subscription = parseSubscription(sub)
	} else if oto, ok := raw["oneTimeProductNotification"].(map[string]any); ok {
		d.Kind = KindOneTime
		d.OneTimeProduct = parseOneTime(oto)
	} else if v, ok := raw["voidedPurchaseNotification"].(map[string]any); ok {
		d.Kind = KindVoided
		d.Voided = parseVoided(v)
	} else if v, ok := raw["testNotification"].(map[string]any); ok {
		d.Kind = KindTest
		d.TestNotification = v
	}
	return d, nil
}

func parseSubscription(m map[string]any) *SubscriptionPayload {
	p := &SubscriptionPayload{}
	if v, ok := m["version"].(string); ok {
		p.Version = v
	}
	if v, ok := m["notificationType"].(float64); ok {
		p.NotificationType = int(v)
		if name, ok := subscriptionNames[int(v)]; ok {
			p.NotificationName = name
		}
	}
	if v, ok := m["purchaseToken"].(string); ok {
		p.PurchaseToken = v
	}
	if v, ok := m["subscriptionId"].(string); ok {
		p.SubscriptionID = v
	}
	return p
}

func parseOneTime(m map[string]any) *OneTimeProductPayload {
	p := &OneTimeProductPayload{}
	if v, ok := m["version"].(string); ok {
		p.Version = v
	}
	if v, ok := m["notificationType"].(float64); ok {
		p.NotificationType = int(v)
		if name, ok := oneTimeNames[int(v)]; ok {
			p.NotificationName = name
		}
	}
	if v, ok := m["purchaseToken"].(string); ok {
		p.PurchaseToken = v
	}
	if v, ok := m["sku"].(string); ok {
		p.SKU = v
	}
	return p
}

func parseVoided(m map[string]any) *VoidedPurchasePayload {
	p := &VoidedPurchasePayload{}
	if v, ok := m["purchaseToken"].(string); ok {
		p.PurchaseToken = v
	}
	if v, ok := m["orderId"].(string); ok {
		p.OrderID = v
	}
	if v, ok := m["productType"].(float64); ok {
		p.ProductType = int(v)
	}
	if v, ok := m["refundType"].(float64); ok {
		p.RefundType = int(v)
	}
	return p
}

// tiny helper to avoid importing bytes just for one conversion.
func bytes(s string) []byte { return []byte(s) }
