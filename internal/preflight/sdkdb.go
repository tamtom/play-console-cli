package preflight

// SDK signature database.
//
// Detection is heuristic: it looks for type descriptors in dex bytecode and
// for characteristic entry paths. Aggressive code shrinking can rename or
// remove classes, so a miss is not proof an SDK is absent. Matches, on the
// other hand, are high confidence — the markers below are package prefixes
// that only the named SDK uses.

// SDK categories.
const (
	categoryBilling  = "billing"
	categoryPayment  = "payment"
	categoryTracking = "tracking"
	categoryAds      = "ads"
)

// sdkSignature identifies one third-party SDK.
type sdkSignature struct {
	ID       string
	Name     string
	Category string
	// Markers are dex type-descriptor prefixes.
	Markers []string
	// Paths are substrings matched against archive entry names.
	Paths []string
}

// Well-known signature IDs referenced directly by scanners.
const (
	sdkPlayBilling = "google_play_billing"
	sdkAdMob       = "admob"
	sdkFirebase    = "firebase_analytics"
)

var sdkSignatures = []sdkSignature{
	// --- In-app billing -----------------------------------------------------
	{
		ID: sdkPlayBilling, Name: "Google Play Billing Library", Category: categoryBilling,
		Markers: []string{"Lcom/android/billingclient/api/"},
	},
	{
		ID: "revenuecat", Name: "RevenueCat", Category: categoryBilling,
		Markers: []string{"Lcom/revenuecat/purchases/"},
	},
	{
		ID: "adapty", Name: "Adapty", Category: categoryBilling,
		Markers: []string{"Lcom/adapty/"},
	},
	{
		ID: "qonversion", Name: "Qonversion", Category: categoryBilling,
		Markers: []string{"Lcom/qonversion/android/sdk/"},
	},
	{
		ID: "amazon_iap", Name: "Amazon In-App Purchasing", Category: categoryBilling,
		Markers: []string{"Lcom/amazon/device/iap/"},
	},
	{
		ID: "huawei_iap", Name: "Huawei IAP", Category: categoryBilling,
		Markers: []string{"Lcom/huawei/hms/iap/"},
	},

	// --- Third-party payment processors ------------------------------------
	{
		ID: "stripe", Name: "Stripe", Category: categoryPayment,
		Markers: []string{"Lcom/stripe/android/"},
	},
	{
		ID: "braintree", Name: "Braintree", Category: categoryPayment,
		Markers: []string{"Lcom/braintreepayments/api/"},
	},
	{
		ID: "paypal", Name: "PayPal", Category: categoryPayment,
		Markers: []string{"Lcom/paypal/checkout/", "Lcom/paypal/android/sdk/"},
	},
	{
		ID: "adyen", Name: "Adyen", Category: categoryPayment,
		Markers: []string{"Lcom/adyen/checkout/"},
	},
	{
		ID: "square", Name: "Square", Category: categoryPayment,
		Markers: []string{"Lcom/squareup/sdk/"},
	},
	{
		ID: "razorpay", Name: "Razorpay", Category: categoryPayment,
		Markers: []string{"Lcom/razorpay/"},
	},
	{
		ID: "paddle", Name: "Paddle", Category: categoryPayment,
		Markers: []string{"Lcom/paddle/android/"},
	},
	{
		ID: "mercadopago", Name: "Mercado Pago", Category: categoryPayment,
		Markers: []string{"Lcom/mercadopago/android/"},
	},
	{
		ID: "flutterwave", Name: "Flutterwave", Category: categoryPayment,
		Markers: []string{"Lcom/flutterwave/raveandroid/"},
	},

	// --- Analytics, attribution, crash reporting ---------------------------
	{
		ID: sdkFirebase, Name: "Firebase Analytics", Category: categoryTracking,
		Markers: []string{"Lcom/google/firebase/analytics/"},
	},
	{
		ID: "crashlytics", Name: "Firebase Crashlytics", Category: categoryTracking,
		Markers: []string{"Lcom/google/firebase/crashlytics/"},
	},
	{
		ID: "facebook", Name: "Meta (Facebook) SDK", Category: categoryTracking,
		Markers: []string{"Lcom/facebook/appevents/", "Lcom/facebook/FacebookSdk;"},
	},
	{
		ID: "appsflyer", Name: "AppsFlyer", Category: categoryTracking,
		Markers: []string{"Lcom/appsflyer/"},
	},
	{
		ID: "adjust", Name: "Adjust", Category: categoryTracking,
		Markers: []string{"Lcom/adjust/sdk/"},
	},
	{
		ID: "branch", Name: "Branch", Category: categoryTracking,
		Markers: []string{"Lio/branch/referral/"},
	},
	{
		ID: "amplitude", Name: "Amplitude", Category: categoryTracking,
		Markers: []string{"Lcom/amplitude/"},
	},
	{
		ID: "mixpanel", Name: "Mixpanel", Category: categoryTracking,
		Markers: []string{"Lcom/mixpanel/android/"},
	},
	{
		ID: "segment", Name: "Segment", Category: categoryTracking,
		Markers: []string{"Lcom/segment/analytics/"},
	},
	{
		ID: "sentry", Name: "Sentry", Category: categoryTracking,
		Markers: []string{"Lio/sentry/android/"},
	},
	{
		ID: "bugsnag", Name: "Bugsnag", Category: categoryTracking,
		Markers: []string{"Lcom/bugsnag/android/"},
	},
	{
		ID: "onesignal", Name: "OneSignal", Category: categoryTracking,
		Markers: []string{"Lcom/onesignal/"},
	},
	{
		ID: "braze", Name: "Braze", Category: categoryTracking,
		Markers: []string{"Lcom/braze/"},
	},
	{
		ID: "airship", Name: "Airship", Category: categoryTracking,
		Markers: []string{"Lcom/urbanairship/"},
	},
	{
		ID: "kochava", Name: "Kochava", Category: categoryTracking,
		Markers: []string{"Lcom/kochava/"},
	},
	{
		ID: "singular", Name: "Singular", Category: categoryTracking,
		Markers: []string{"Lcom/singular/sdk/"},
	},
	{
		ID: "flurry", Name: "Flurry", Category: categoryTracking,
		Markers: []string{"Lcom/flurry/android/"},
	},
	{
		ID: "umeng", Name: "Umeng", Category: categoryTracking,
		Markers: []string{"Lcom/umeng/"},
	},
	{
		ID: "tenjin", Name: "Tenjin", Category: categoryTracking,
		Markers: []string{"Lcom/tenjin/android/"},
	},
	{
		ID: "clevertap", Name: "CleverTap", Category: categoryTracking,
		Markers: []string{"Lcom/clevertap/android/sdk/"},
	},
	{
		ID: "moengage", Name: "MoEngage", Category: categoryTracking,
		Markers: []string{"Lcom/moengage/"},
	},
	{
		ID: "newrelic", Name: "New Relic", Category: categoryTracking,
		Markers: []string{"Lcom/newrelic/agent/android/"},
	},
	{
		ID: "datadog", Name: "Datadog", Category: categoryTracking,
		Markers: []string{"Lcom/datadog/android/"},
	},
	{
		ID: "posthog", Name: "PostHog", Category: categoryTracking,
		Markers: []string{"Lcom/posthog/"},
	},
	{
		ID: "countly", Name: "Countly", Category: categoryTracking,
		Markers: []string{"Lly/count/android/sdk/"},
	},
	{
		ID: "yandex_metrica", Name: "Yandex AppMetrica", Category: categoryTracking,
		Markers: []string{"Lcom/yandex/metrica/", "Lio/appmetrica/analytics/"},
	},

	// --- Advertising --------------------------------------------------------
	{
		ID: sdkAdMob, Name: "Google Mobile Ads (AdMob)", Category: categoryAds,
		Markers: []string{"Lcom/google/android/gms/ads/"},
	},
	{
		ID: "applovin", Name: "AppLovin", Category: categoryAds,
		Markers: []string{"Lcom/applovin/"},
	},
	{
		ID: "ironsource", Name: "ironSource / Unity LevelPlay", Category: categoryAds,
		Markers: []string{"Lcom/ironsource/"},
	},
	{
		ID: "unity_ads", Name: "Unity Ads", Category: categoryAds,
		Markers: []string{"Lcom/unity3d/ads/"},
	},
	{
		ID: "vungle", Name: "Vungle / Liftoff", Category: categoryAds,
		Markers: []string{"Lcom/vungle/"},
	},
	{
		ID: "chartboost", Name: "Chartboost", Category: categoryAds,
		Markers: []string{"Lcom/chartboost/"},
	},
	{
		ID: "inmobi", Name: "InMobi", Category: categoryAds,
		Markers: []string{"Lcom/inmobi/"},
	},
	{
		ID: "pangle", Name: "Pangle (ByteDance)", Category: categoryAds,
		Markers: []string{"Lcom/bytedance/sdk/openadsdk/"},
	},
	{
		ID: "mintegral", Name: "Mintegral", Category: categoryAds,
		Markers: []string{"Lcom/mbridge/msdk/"},
	},
	{
		ID: "fyber", Name: "Fyber / Digital Turbine", Category: categoryAds,
		Markers: []string{"Lcom/fyber/"},
	},
	{
		ID: "moloco", Name: "Moloco", Category: categoryAds,
		Markers: []string{"Lcom/moloco/sdk/"},
	},
	{
		ID: "meta_audience", Name: "Meta Audience Network", Category: categoryAds,
		Markers: []string{"Lcom/facebook/ads/"},
	},
	{
		ID: "tapjoy", Name: "Tapjoy", Category: categoryAds,
		Markers: []string{"Lcom/tapjoy/"},
	},
	{
		ID: "adcolony", Name: "AdColony", Category: categoryAds,
		Markers: []string{"Lcom/adcolony/sdk/"},
	},
	{
		ID: "smaato", Name: "Smaato", Category: categoryAds,
		Markers: []string{"Lcom/smaato/sdk/"},
	},
}

// sdkNameByID resolves a signature ID to its display name.
func sdkNameByID(id string) string {
	for _, s := range sdkSignatures {
		if s.ID == id {
			return s.Name
		}
	}
	return id
}
