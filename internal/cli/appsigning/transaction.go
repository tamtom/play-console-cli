package appsigning

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/appsigningclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

const (
	signingTransactionVersion = 1
	operationEnroll           = "enroll"
	operationRotate           = "rotate-key"
	receiptPlanned            = "planned"
	receiptInProgress         = "in-progress"
	receiptComplete           = "complete"
	receiptRejected           = "rejected"
	receiptAmbiguous          = "ambiguous"
)

type signingPlan struct {
	Version       int             `json:"version"`
	Provider      string          `json:"provider"`
	API           string          `json:"api"`
	Operation     string          `json:"operation"`
	Package       string          `json:"package"`
	CreatedAt     string          `json:"createdAt"`
	ExpiresAt     string          `json:"expiresAt"`
	Request       json.RawMessage `json:"request"`
	RequestSHA256 string          `json:"requestSha256"`
	PlanHash      string          `json:"planHash"`
}

type signingReceipt struct {
	Version     int             `json:"version"`
	Provider    string          `json:"provider"`
	PlanHash    string          `json:"planHash"`
	Operation   string          `json:"operation"`
	Package     string          `json:"package"`
	Status      string          `json:"status"`
	StartedAt   string          `json:"startedAt,omitempty"`
	CompletedAt string          `json:"completedAt,omitempty"`
	Response    json.RawMessage `json:"response,omitempty"`
	Error       string          `json:"error,omitempty"`
	ReceiptHash string          `json:"receiptHash"`
}

func createSigningPlan(ctx context.Context, operation, packageName string, request any) (*signingPlan, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode app-signing request: %w", err)
	}
	now := shared.Now(ctx).UTC()
	plan := &signingPlan{
		Version: signingTransactionVersion, Provider: "official-api", API: "android-publisher-v3",
		Operation: operation, Package: packageName, CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(24 * time.Hour).Format(time.RFC3339), Request: encoded,
		RequestSHA256: canonicalSigningRequestHash(encoded),
	}
	plan.PlanHash, err = signingPlanHash(plan)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func writeSigningPlan(filesystem shared.Filesystem, path string, plan *signingPlan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app-signing plan: %w", err)
	}
	if err := filesystem.AtomicWriteFile(path, data, 0o600, 0o700); err != nil {
		return fmt.Errorf("write app-signing plan: %w", err)
	}
	return nil
}

// ApplyCommand executes one exact sealed enterprise App Signing plan.
func ApplyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-signing apply", flag.ExitOnError)
	planFile := fs.String("plan-file", "", "Path to the sealed operation plan")
	receiptFile := fs.String("receipt-file", "", "Path for the sealed execution receipt")
	confirmPlan := fs.String("confirm-plan", "", "Exact SHA-256 plan ID to authorize")
	enterprise := fs.Bool("enterprise-self-hosted-kms", false, "Acknowledge this app uses the enterprise self-hosted Cloud KMS program")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "apply",
		ShortUsage: "gplay app-signing apply --plan-file <path> --receipt-file <path> --confirm-plan <sha256> --enterprise-self-hosted-kms",
		ShortHelp:  "Apply an exact sealed enterprise App Signing plan once.",
		LongHelp: `A sealed in-progress receipt is written before the irreversible request. If
the transport result is ambiguous, the receipt blocks automatic replay because
Google publishes no App Signing readback endpoint. Resolve that state with
Google support or Play Console before creating another plan.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			if !*enterprise {
				return shared.UsageError("--enterprise-self-hosted-kms is required")
			}
			if strings.TrimSpace(*planFile) == "" || strings.TrimSpace(*receiptFile) == "" || strings.TrimSpace(*confirmPlan) == "" {
				return shared.UsageError("--plan-file, --receipt-file, and --confirm-plan are required")
			}
			receipt, err := applySigningPlan(ctx, *planFile, *receiptFile, strings.TrimSpace(*confirmPlan))
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, receipt, *output, *pretty)
		},
	}
}

func applySigningPlan(ctx context.Context, planPath, receiptPath, confirmation string) (*signingReceipt, error) {
	filesystem := shared.FilesystemFrom(ctx)
	plan, err := readSigningPlan(filesystem, planPath)
	if err != nil {
		return nil, err
	}
	if confirmation != plan.PlanHash {
		return nil, fmt.Errorf("--confirm-plan must exactly match plan ID %s", plan.PlanHash)
	}
	receipt, exists, err := readSigningReceipt(filesystem, receiptPath, plan)
	if err != nil {
		return nil, err
	}
	if exists {
		switch receipt.Status {
		case receiptComplete:
			return receipt, nil
		case receiptInProgress, receiptAmbiguous:
			return nil, fmt.Errorf("app-signing receipt is %s and must not be replayed automatically; reconcile the operation manually", receipt.Status)
		case receiptRejected:
			return nil, fmt.Errorf("app-signing receipt is rejected; inspect it and create a new plan before retrying")
		default:
			return nil, fmt.Errorf("app-signing receipt has unsupported status %q", receipt.Status)
		}
	}
	expiresAt, err := time.Parse(time.RFC3339, plan.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("app-signing plan has invalid expiry: %w", err)
	}
	if !shared.Now(ctx).Before(expiresAt) {
		return nil, fmt.Errorf("app-signing plan expired at %s", plan.ExpiresAt)
	}
	if shared.IsDryRun(ctx) {
		planned := newSigningReceipt(ctx, plan, receiptPlanned)
		if err := sealSigningReceipt(planned); err != nil {
			return nil, err
		}
		return planned, nil
	}

	receipt = newSigningReceipt(ctx, plan, receiptInProgress)
	if err := createSigningReceiptExclusive(filesystem, receiptPath, receipt); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("app-signing operation is already reserved by another process; inspect receipt %s", receiptPath)
		}
		return nil, err
	}
	service, err := appsigningclient.NewService(ctx)
	if err != nil {
		receipt.Status = receiptRejected
		receipt.Error = fmt.Sprintf("pre-submit service setup failed: %v", err)
		if writeErr := writeSigningReceipt(filesystem, receiptPath, receipt); writeErr != nil {
			return nil, errors.Join(err, writeErr)
		}
		return nil, fmt.Errorf("create app-signing service before submission: %w", err)
	}
	response, applyErr := executeSigningPlan(ctx, service, plan)
	if applyErr != nil {
		if isDefinitiveAppSigningRejection(applyErr) {
			receipt.Status = receiptRejected
		} else {
			receipt.Status = receiptAmbiguous
		}
		receipt.Error = applyErr.Error()
		if writeErr := writeSigningReceipt(filesystem, receiptPath, receipt); writeErr != nil {
			return nil, errors.Join(applyErr, writeErr)
		}
		return nil, fmt.Errorf("app-signing apply is %s and must not be replayed automatically: %w", receipt.Status, applyErr)
	}
	encodedResponse, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode app-signing response: %w", err)
	}
	receipt.Status = receiptComplete
	receipt.CompletedAt = shared.Now(ctx).UTC().Format(time.RFC3339)
	receipt.Response = encodedResponse
	if err := writeSigningReceipt(filesystem, receiptPath, receipt); err != nil {
		return nil, err
	}
	return receipt, nil
}

func isDefinitiveAppSigningRejection(err error) bool {
	var statusErr *appsigningclient.HTTPStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	switch statusErr.StatusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestURITooLong,
		http.StatusUnsupportedMediaType,
		http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}

func executeSigningPlan(ctx context.Context, service *appsigningclient.Service, plan *signingPlan) (any, error) {
	requestCtx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
	defer cancel()
	switch plan.Operation {
	case operationEnroll:
		var request appsigningclient.EnrollAppRequest
		if err := json.Unmarshal(plan.Request, &request); err != nil {
			return nil, fmt.Errorf("decode enrollment request: %w", err)
		}
		return service.API.EnrollApp(requestCtx, plan.Package, &request)
	case operationRotate:
		var request appsigningclient.RotateAppSigningKeyRequest
		if err := json.Unmarshal(plan.Request, &request); err != nil {
			return nil, fmt.Errorf("decode rotation request: %w", err)
		}
		return service.API.RotateAppSigningKey(requestCtx, plan.Package, &request)
	default:
		return nil, fmt.Errorf("unsupported app-signing operation %q", plan.Operation)
	}
}

func readSigningPlan(filesystem shared.Filesystem, path string) (*signingPlan, error) {
	data, err := filesystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read app-signing plan: %w", err)
	}
	var plan signingPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("decode app-signing plan: %w", err)
	}
	if plan.Version != signingTransactionVersion || plan.Provider != "official-api" || plan.API != "android-publisher-v3" {
		return nil, fmt.Errorf("unsupported app-signing plan contract")
	}
	if plan.Operation != operationEnroll && plan.Operation != operationRotate {
		return nil, fmt.Errorf("unsupported app-signing operation %q", plan.Operation)
	}
	if !signingPackagePattern.MatchString(plan.Package) {
		return nil, fmt.Errorf("app-signing plan has invalid package")
	}
	if canonicalSigningRequestHash(plan.Request) != plan.RequestSHA256 {
		return nil, fmt.Errorf("app-signing request hash mismatch")
	}
	expected, err := signingPlanHash(&plan)
	if err != nil {
		return nil, err
	}
	if plan.PlanHash != expected {
		return nil, fmt.Errorf("app-signing plan hash mismatch: got %s, expected %s", plan.PlanHash, expected)
	}
	if err := validateSigningPlanRequest(&plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func validateSigningPlanRequest(plan *signingPlan) error {
	switch plan.Operation {
	case operationEnroll:
		var request appsigningclient.EnrollAppRequest
		if err := json.Unmarshal(plan.Request, &request); err != nil {
			return fmt.Errorf("decode enrollment request: %w", err)
		}
		return validateEnrollRequest(&request)
	case operationRotate:
		var request appsigningclient.RotateAppSigningKeyRequest
		if err := json.Unmarshal(plan.Request, &request); err != nil {
			return fmt.Errorf("decode rotation request: %w", err)
		}
		return validateRotateRequest(&request)
	default:
		return fmt.Errorf("unsupported app-signing operation %q", plan.Operation)
	}
}

func newSigningReceipt(ctx context.Context, plan *signingPlan, status string) *signingReceipt {
	return &signingReceipt{
		Version: signingTransactionVersion, Provider: plan.Provider, PlanHash: plan.PlanHash,
		Operation: plan.Operation, Package: plan.Package, Status: status,
		StartedAt: shared.Now(ctx).UTC().Format(time.RFC3339),
	}
}

func readSigningReceipt(filesystem shared.Filesystem, path string, plan *signingPlan) (*signingReceipt, bool, error) {
	data, err := filesystem.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read app-signing receipt: %w", err)
	}
	var receipt signingReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, false, fmt.Errorf("decode app-signing receipt: %w", err)
	}
	if receipt.Version != signingTransactionVersion || receipt.Provider != plan.Provider || receipt.PlanHash != plan.PlanHash || receipt.Operation != plan.Operation || receipt.Package != plan.Package {
		return nil, false, fmt.Errorf("app-signing receipt does not belong to plan %s", plan.PlanHash)
	}
	expected, err := signingReceiptHash(&receipt)
	if err != nil {
		return nil, false, err
	}
	if receipt.ReceiptHash != expected {
		return nil, false, fmt.Errorf("app-signing receipt hash mismatch")
	}
	return &receipt, true, nil
}

func writeSigningReceipt(filesystem shared.Filesystem, path string, receipt *signingReceipt) error {
	if err := sealSigningReceipt(receipt); err != nil {
		return err
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app-signing receipt: %w", err)
	}
	if err := filesystem.AtomicWriteFile(path, data, 0o600, 0o700); err != nil {
		return fmt.Errorf("write app-signing receipt: %w", err)
	}
	return nil
}

func createSigningReceiptExclusive(filesystem shared.Filesystem, path string, receipt *signingReceipt) error {
	if err := sealSigningReceipt(receipt); err != nil {
		return err
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app-signing reservation receipt: %w", err)
	}
	if err := filesystem.CreateExclusiveFile(path, data, 0o600, 0o700); err != nil {
		return fmt.Errorf("reserve app-signing operation: %w", err)
	}
	return nil
}

func sealSigningReceipt(receipt *signingReceipt) error {
	hash, err := signingReceiptHash(receipt)
	if err != nil {
		return err
	}
	receipt.ReceiptHash = hash
	return nil
}

func signingPlanHash(plan *signingPlan) (string, error) {
	copyPlan := *plan
	copyPlan.PlanHash = ""
	copyPlan.Request = canonicalSigningRequest(copyPlan.Request)
	return hashSigningValue(copyPlan)
}

func signingReceiptHash(receipt *signingReceipt) (string, error) {
	copyReceipt := *receipt
	copyReceipt.ReceiptHash = ""
	return hashSigningValue(copyReceipt)
}

func hashSigningValue(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode app-signing hash input: %w", err)
	}
	return sha256Text(data), nil
}

func sha256Text(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func canonicalSigningRequest(data []byte) json.RawMessage {
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, data); err != nil {
		return append(json.RawMessage(nil), data...)
	}
	return compacted.Bytes()
}

func canonicalSigningRequestHash(data []byte) string {
	return sha256Text(canonicalSigningRequest(data))
}
